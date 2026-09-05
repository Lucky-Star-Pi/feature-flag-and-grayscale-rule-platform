package db

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestMapUniqueViolation_FlagKey(t *testing.T) {
	err := MapUniqueViolation(&pgconn.PgError{
		Code:           "23505",
		ConstraintName: "flags_key_environment_uk",
	})
	require.ErrorIs(t, err, ErrFlagKeyConflict)
}

func TestMapUniqueViolation_RulePriority(t *testing.T) {
	err := MapUniqueViolation(&pgconn.PgError{
		Code:           "23505",
		ConstraintName: "rules_flag_id_priority_uk",
	})
	require.ErrorIs(t, err, ErrRulePriorityConflict)
}

func TestMapUniqueViolation_Passthrough(t *testing.T) {
	orig := errors.New("other")
	require.Equal(t, orig, MapUniqueViolation(orig))
	require.Nil(t, MapUniqueViolation(nil))
}
