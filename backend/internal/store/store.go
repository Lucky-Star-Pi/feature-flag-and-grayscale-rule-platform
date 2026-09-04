package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"featureflag/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrFlagKeyConflict  = errors.New("flag key conflict")
	ErrRulePriorityConflict = errors.New("rule priority conflict")
)

type Store struct {
	DB *sqlx.DB
}

func Open(dsn string) (*Store, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) BeginTxx(ctx context.Context) (*sqlx.Tx, error) {
	return s.DB.BeginTxx(ctx, nil)
}

func MapPGError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch {
		case strings.Contains(pgErr.ConstraintName, "flags_environment_key"):
			return ErrFlagKeyConflict
		case strings.Contains(pgErr.ConstraintName, "flag_rules_flag_id_priority"):
			return ErrRulePriorityConflict
		default:
			return err
		}
	}
	return err
}

func (s *Store) ListFlags(ctx context.Context, q, env string, enabled *bool) ([]model.Flag, error) {
	var flags []model.Flag
	query := `SELECT id, name, key, environment, enabled, default_value, created_at, updated_at FROM flags WHERE 1=1`
	args := []any{}
	n := 1
	if q != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR key ILIKE $%d)`, n, n)
		args = append(args, "%"+q+"%")
		n++
	}
	if env != "" {
		query += fmt.Sprintf(` AND environment = $%d`, n)
		args = append(args, env)
		n++
	}
	if enabled != nil {
		query += fmt.Sprintf(` AND enabled = $%d`, n)
		args = append(args, *enabled)
	}
	query += ` ORDER BY updated_at DESC`
	if err := s.DB.SelectContext(ctx, &flags, query, args...); err != nil {
		return nil, err
	}
	if flags == nil {
		flags = []model.Flag{}
	}
	return flags, nil
}

func (s *Store) GetFlag(ctx context.Context, id int64) (*model.Flag, error) {
	var f model.Flag
	err := s.DB.GetContext(ctx, &f, `SELECT id, name, key, environment, enabled, default_value, created_at, updated_at FROM flags WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) GetFlagByKeyEnv(ctx context.Context, key, env string) (*model.Flag, error) {
	var f model.Flag
	err := s.DB.GetContext(ctx, &f, `SELECT id, name, key, environment, enabled, default_value, created_at, updated_at FROM flags WHERE key=$1 AND environment=$2`, key, env)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) InsertFlagTx(ctx context.Context, tx *sqlx.Tx, f *model.Flag) error {
	err := tx.QueryRowxContext(ctx, `
		INSERT INTO flags (name, key, environment, enabled, default_value)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at`,
		f.Name, f.Key, f.Environment, f.Enabled, f.DefaultValue,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	return MapPGError(err)
}

func (s *Store) UpdateFlagTx(ctx context.Context, tx *sqlx.Tx, id int64, name string, defaultValue bool) (*model.Flag, error) {
	var f model.Flag
	err := tx.QueryRowxContext(ctx, `
		UPDATE flags SET name=$1, default_value=$2, updated_at=NOW()
		WHERE id=$3
		RETURNING id, name, key, environment, enabled, default_value, created_at, updated_at`,
		name, defaultValue, id,
	).StructScan(&f)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

func (s *Store) SetEnabledTx(ctx context.Context, tx *sqlx.Tx, id int64, enabled bool) (*model.Flag, error) {
	var f model.Flag
	err := tx.QueryRowxContext(ctx, `
		UPDATE flags SET enabled=$1, updated_at=NOW()
		WHERE id=$2
		RETURNING id, name, key, environment, enabled, default_value, created_at, updated_at`,
		enabled, id,
	).StructScan(&f)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

func (s *Store) ListRules(ctx context.Context, flagID int64) ([]model.Rule, error) {
	var rules []model.Rule
	err := s.DB.SelectContext(ctx, &rules, `
		SELECT id, flag_id, attribute, operator, expected_value, return_value, priority, created_at, updated_at
		FROM flag_rules WHERE flag_id=$1 ORDER BY priority ASC, id ASC`, flagID)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []model.Rule{}
	}
	return rules, nil
}

func (s *Store) GetRule(ctx context.Context, flagID, ruleID int64) (*model.Rule, error) {
	var r model.Rule
	err := s.DB.GetContext(ctx, &r, `
		SELECT id, flag_id, attribute, operator, expected_value, return_value, priority, created_at, updated_at
		FROM flag_rules WHERE id=$1 AND flag_id=$2`, ruleID, flagID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) InsertRuleTx(ctx context.Context, tx *sqlx.Tx, r *model.Rule) error {
	err := tx.QueryRowxContext(ctx, `
		INSERT INTO flag_rules (flag_id, attribute, operator, expected_value, return_value, priority)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		r.FlagID, r.Attribute, r.Operator, r.ExpectedValue, r.ReturnValue, r.Priority,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	return MapPGError(err)
}

func (s *Store) UpdateRuleTx(ctx context.Context, tx *sqlx.Tx, r *model.Rule) error {
	err := tx.QueryRowxContext(ctx, `
		UPDATE flag_rules SET attribute=$1, operator=$2, expected_value=$3, return_value=$4, priority=$5, updated_at=NOW()
		WHERE id=$6 AND flag_id=$7
		RETURNING id, created_at, updated_at`,
		r.Attribute, r.Operator, r.ExpectedValue, r.ReturnValue, r.Priority, r.ID, r.FlagID,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return MapPGError(err)
}

func (s *Store) DeleteRuleTx(ctx context.Context, tx *sqlx.Tx, flagID, ruleID int64) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM flag_rules WHERE id=$1 AND flag_id=$2`, ruleID, flagID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) PriorityExists(ctx context.Context, flagID int64, priority int, excludeRuleID int64) (bool, error) {
	var exists bool
	err := s.DB.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM flag_rules WHERE flag_id=$1 AND priority=$2 AND id <> $3
		)`, flagID, priority, excludeRuleID)
	return exists, err
}

func (s *Store) InsertHistoryTx(ctx context.Context, tx *sqlx.Tx, h *model.History) error {
	return tx.QueryRowxContext(ctx, `
		INSERT INTO operation_histories (flag_id, actor, action, summary, payload)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		h.FlagID, h.Actor, h.Action, h.Summary, h.Payload,
	).Scan(&h.ID, &h.CreatedAt)
}

func (s *Store) ListHistories(ctx context.Context, flagID int64) ([]model.History, error) {
	var rows []model.History
	err := s.DB.SelectContext(ctx, &rows, `
		SELECT id, flag_id, actor, action, summary, payload, created_at
		FROM operation_histories WHERE flag_id=$1 ORDER BY created_at DESC, id DESC`, flagID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.History{}
	}
	return rows, nil
}

func (s *Store) CountFlags(ctx context.Context) (int, error) {
	var n int
	err := s.DB.GetContext(ctx, &n, `SELECT COUNT(*) FROM flags`)
	return n, err
}

func (s *Store) CountHistoriesByFlag(ctx context.Context, flagID int64) (int, error) {
	var n int
	err := s.DB.GetContext(ctx, &n, `SELECT COUNT(*) FROM operation_histories WHERE flag_id=$1`, flagID)
	return n, err
}
