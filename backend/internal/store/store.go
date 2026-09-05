package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"featureflag/internal/db"
	"featureflag/internal/model"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("not found")

const flagColumns = `id, name, key, environment, enabled, default_value, created_at, updated_at`
const ruleColumns = `id, flag_id, attribute, operator, expected_value, return_value, priority, created_at, updated_at`
const historyColumns = `id, flag_id, operation_type, operator, summary, created_at`

type FlagFilter struct {
	Key         string
	Environment string
	Enabled     *bool
}

func ListFlags(ctx context.Context, q sqlx.ExtContext, f FlagFilter) ([]model.Flag, error) {
	query := `SELECT ` + flagColumns + ` FROM flags WHERE 1=1`
	args := []any{}
	n := 1
	if f.Key != "" {
		query += fmt.Sprintf(` AND key ILIKE $%d`, n)
		args = append(args, "%"+f.Key+"%")
		n++
	}
	if f.Environment != "" {
		query += fmt.Sprintf(` AND environment = $%d`, n)
		args = append(args, f.Environment)
		n++
	}
	if f.Enabled != nil {
		query += fmt.Sprintf(` AND enabled = $%d`, n)
		args = append(args, *f.Enabled)
	}
	query += ` ORDER BY updated_at DESC`
	var flags []model.Flag
	if err := sqlx.SelectContext(ctx, q, &flags, query, args...); err != nil {
		return nil, err
	}
	if flags == nil {
		flags = []model.Flag{}
	}
	return flags, nil
}

func GetFlag(ctx context.Context, q sqlx.ExtContext, id int64) (*model.Flag, error) {
	var f model.Flag
	err := sqlx.GetContext(ctx, q, &f, `SELECT `+flagColumns+` FROM flags WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func GetFlagByKeyAndEnv(ctx context.Context, q sqlx.ExtContext, key, env string) (*model.Flag, error) {
	var f model.Flag
	err := sqlx.GetContext(ctx, q, &f, `SELECT `+flagColumns+` FROM flags WHERE key=$1 AND environment=$2`, key, env)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func CreateFlag(ctx context.Context, tx *sqlx.Tx, f *model.Flag) error {
	err := tx.QueryRowxContext(ctx, `
		INSERT INTO flags (name, key, environment, enabled, default_value)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING `+flagColumns,
		f.Name, f.Key, f.Environment, f.Enabled, f.DefaultValue,
	).StructScan(f)
	return db.MapUniqueViolation(err)
}

func UpdateFlag(ctx context.Context, tx *sqlx.Tx, id int64, name string, defaultValue bool) (*model.Flag, error) {
	var f model.Flag
	err := tx.QueryRowxContext(ctx, `
		UPDATE flags SET name=$1, default_value=$2, updated_at=NOW()
		WHERE id=$3
		RETURNING `+flagColumns,
		name, defaultValue, id,
	).StructScan(&f)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, db.MapUniqueViolation(err)
}

func SetFlagEnabled(ctx context.Context, tx *sqlx.Tx, id int64, enabled bool) (*model.Flag, error) {
	var f model.Flag
	err := tx.QueryRowxContext(ctx, `
		UPDATE flags SET enabled=$1, updated_at=NOW()
		WHERE id=$2
		RETURNING `+flagColumns,
		enabled, id,
	).StructScan(&f)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

func ListRules(ctx context.Context, q sqlx.ExtContext, flagID int64) ([]model.Rule, error) {
	var rules []model.Rule
	err := sqlx.SelectContext(ctx, q, &rules, `
		SELECT `+ruleColumns+`
		FROM rules WHERE flag_id=$1
		ORDER BY priority ASC, id ASC`, flagID)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []model.Rule{}
	}
	return rules, nil
}

func GetRule(ctx context.Context, q sqlx.ExtContext, flagID, ruleID int64) (*model.Rule, error) {
	var r model.Rule
	err := sqlx.GetContext(ctx, q, &r, `
		SELECT `+ruleColumns+` FROM rules WHERE id=$1 AND flag_id=$2`, ruleID, flagID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func PriorityExists(ctx context.Context, q sqlx.ExtContext, flagID int64, priority int, excludeRuleID int64) (bool, error) {
	var exists bool
	err := sqlx.GetContext(ctx, q, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM rules WHERE flag_id=$1 AND priority=$2 AND id <> $3
		)`, flagID, priority, excludeRuleID)
	return exists, err
}

func CreateRule(ctx context.Context, tx *sqlx.Tx, r *model.Rule) error {
	err := tx.QueryRowxContext(ctx, `
		INSERT INTO rules (flag_id, attribute, operator, expected_value, return_value, priority)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+ruleColumns,
		r.FlagID, r.Attribute, r.Operator, r.ExpectedValue, r.ReturnValue, r.Priority,
	).StructScan(r)
	return db.MapUniqueViolation(err)
}

func UpdateRule(ctx context.Context, tx *sqlx.Tx, r *model.Rule) error {
	err := tx.QueryRowxContext(ctx, `
		UPDATE rules
		SET attribute=$1, operator=$2, expected_value=$3, return_value=$4, priority=$5, updated_at=NOW()
		WHERE id=$6 AND flag_id=$7
		RETURNING `+ruleColumns,
		r.Attribute, r.Operator, r.ExpectedValue, r.ReturnValue, r.Priority, r.ID, r.FlagID,
	).StructScan(r)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return db.MapUniqueViolation(err)
}

func DeleteRule(ctx context.Context, tx *sqlx.Tx, flagID, ruleID int64) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM rules WHERE id=$1 AND flag_id=$2`, ruleID, flagID)
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

func InsertHistory(ctx context.Context, tx *sqlx.Tx, h *model.History) error {
	return tx.QueryRowxContext(ctx, `
		INSERT INTO history (flag_id, operation_type, operator, summary)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at`,
		h.FlagID, h.OperationType, h.Operator, h.Summary,
	).Scan(&h.ID, &h.CreatedAt)
}

func ListHistory(ctx context.Context, q sqlx.ExtContext, flagID int64) ([]model.History, error) {
	var rows []model.History
	err := sqlx.SelectContext(ctx, q, &rows, `
		SELECT `+historyColumns+`
		FROM history WHERE flag_id=$1
		ORDER BY created_at DESC, id DESC`, flagID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.History{}
	}
	return rows, nil
}
