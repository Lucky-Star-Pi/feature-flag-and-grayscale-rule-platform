package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// 业务冲突错误（由 HTTP 层集中映射）：
//   - ErrFlagKeyConflict      → 409 KEY_CONFLICT
//   - ErrRulePriorityConflict → 400 PRIORITY_CONFLICT（重复优先级视为输入错误，非 409）
//   - ErrVersionConflict      → 409 VERSION_CONFLICT（编辑乐观锁未命中）
var (
	ErrFlagKeyConflict      = errors.New("flag key conflict in same environment")
	ErrRulePriorityConflict = errors.New("rule priority conflict for same flag")
	ErrVersionConflict      = errors.New("concurrent modification: version mismatch")
)

// DB 包装 sqlx，供 M2 CRUD 使用。
type DB struct {
	SQL *sqlx.DB
}

func Open(dsn string) (*DB, error) {
	sqlDB, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	return &DB{SQL: sqlDB}, nil
}

func (d *DB) Close() error {
	return d.SQL.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	return d.SQL.PingContext(ctx)
}

// WithTx 在同一事务中执行 fn；成功 Commit，失败 Rollback。
// M2 起：业务变更与 history 写入必须全部放在 fn 内。
func (d *DB) WithTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := d.SQL.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// MapUniqueViolation 将 PostgreSQL unique_violation(23505) 转为业务冲突错误。
func MapUniqueViolation(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		name := pgErr.ConstraintName
		switch {
		case strings.Contains(name, "flags_key_environment"):
			return ErrFlagKeyConflict
		case strings.Contains(name, "rules_flag_id_priority"):
			return ErrRulePriorityConflict
		default:
			return fmt.Errorf("unique constraint violated (%s): %w", name, err)
		}
	}
	return err
}
