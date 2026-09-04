package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"featureflag/internal/model"
	"featureflag/internal/service"
	"featureflag/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skip integration tests")
	}
	return dsn
}

func setupService(t *testing.T) (*service.Service, *store.Store) {
	t.Helper()
	dsn := testDSN(t)
	migPath := migrationsFileURL(t)
	m, err := migrate.New(migPath, dsn)
	require.NoError(t, err)
	_ = m.Down()
	require.NoError(t, m.Up())
	ver, dirty, err := m.Version()
	require.NoError(t, err)
	require.False(t, dirty)
	require.GreaterOrEqual(t, int(ver), 1)
	m.Close()

	st, err := store.Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return service.New(st), st
}

func migrationsFileURL(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// service_test lives in internal/service → ../../migrations
	p := filepath.Clean(filepath.Join(wd, "..", "..", "migrations"))
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return "file://" + filepath.ToSlash(abs)
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestFlagKeyUniqueConstraint(t *testing.T) {
	svc, st := setupService(t)
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "A", Key: "pay", Environment: "development", DefaultValue: boolPtr(false),
	})
	require.NoError(t, err)

	_, err = svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "B", Key: "pay", Environment: "development", DefaultValue: boolPtr(true),
	})
	require.ErrorIs(t, err, store.ErrFlagKeyConflict)

	_, err = svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "C", Key: "pay", Environment: "production", DefaultValue: boolPtr(false),
	})
	require.NoError(t, err)

	n, err := st.CountFlags(ctx)
	require.NoError(t, err)
	// seed has 3 flags + 2 created = 5 (seed re-applied) OR if we Down/Up seed is there
	// After Down+Up: seed 3 + pay@dev + pay@prod = 5
	require.Equal(t, 5, n)
}

func TestHistoryAtomicRollback(t *testing.T) {
	svc, st := setupService(t)
	ctx := context.Background()
	before, err := st.CountFlags(ctx)
	require.NoError(t, err)

	svc.FailHistoryInsert = true
	_, err = svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "Rollback", Key: "rollback_key", Environment: "staging", DefaultValue: boolPtr(false),
	})
	require.Error(t, err)

	after, err := st.CountFlags(ctx)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestDuplicatePriorityRejected(t *testing.T) {
	svc, _ := setupService(t)
	ctx := context.Background()
	f, err := svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "R", Key: "rule_flag", Environment: "development", DefaultValue: boolPtr(false),
	})
	require.NoError(t, err)

	_, err = svc.CreateRule(ctx, f.ID, service.RuleInput{
		Attribute: "country", Operator: "equals", ExpectedValue: "CN",
		ReturnValue: boolPtr(true), Priority: intPtr(0),
	})
	require.NoError(t, err)

	_, err = svc.CreateRule(ctx, f.ID, service.RuleInput{
		Attribute: "plan", Operator: "equals", ExpectedValue: "pro",
		ReturnValue: boolPtr(true), Priority: intPtr(0),
	})
	require.ErrorIs(t, err, service.ErrDuplicatePriority)
}

func TestRuleCRUDHistory(t *testing.T) {
	svc, st := setupService(t)
	ctx := context.Background()
	f, err := svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "H", Key: "hist_flag", Environment: "development", DefaultValue: boolPtr(false),
	})
	require.NoError(t, err)

	r, err := svc.CreateRule(ctx, f.ID, service.RuleInput{
		Attribute: "country", Operator: "equals", ExpectedValue: "CN",
		ReturnValue: boolPtr(true), Priority: intPtr(1),
	})
	require.NoError(t, err)

	_, err = svc.UpdateRule(ctx, f.ID, r.ID, service.RuleInput{
		Attribute: "country", Operator: "equals", ExpectedValue: "US",
		ReturnValue: boolPtr(false), Priority: intPtr(1),
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteRule(ctx, f.ID, r.ID))

	histories, err := st.ListHistories(ctx, f.ID)
	require.NoError(t, err)
	actions := map[string]bool{}
	for _, h := range histories {
		actions[h.Action] = true
		require.Equal(t, model.ActorLocalAdmin, h.Actor)
		require.NotEmpty(t, h.Summary)
	}
	require.True(t, actions[model.ActionCreateRule])
	require.True(t, actions[model.ActionUpdateRule])
	require.True(t, actions[model.ActionDeleteRule])
}

func TestEvaluateDisabledAndErrors(t *testing.T) {
	svc, _ := setupService(t)
	ctx := context.Background()

	// seed checkout_v2@production is disabled
	out, err := svc.Evaluate(ctx, service.EvaluateInput{
		Key: "checkout_v2", Environment: "production", Attributes: map[string]any{"country": "CN"},
	})
	require.NoError(t, err)
	require.False(t, out.FinalValue)
	require.False(t, out.Matched)
	require.Equal(t, model.ReasonFlagDisabled, out.Reason)
	require.Nil(t, out.MatchedRule)

	_, err = svc.Evaluate(ctx, service.EvaluateInput{
		Key: "no_such", Environment: "development", Attributes: map[string]any{},
	})
	require.ErrorIs(t, err, service.ErrFlagNotFound)

	_, err = svc.Evaluate(ctx, service.EvaluateInput{
		Key: "dark_mode", Environment: "development", Attributes: nil,
	})
	require.ErrorIs(t, err, service.ErrInvalidJSON)
}

func TestEnableDisableIdempotent(t *testing.T) {
	svc, st := setupService(t)
	ctx := context.Background()
	f, err := svc.CreateFlag(ctx, service.CreateFlagInput{
		Name: "E", Key: "en_flag", Environment: "development", Enabled: boolPtr(true), DefaultValue: boolPtr(false),
	})
	require.NoError(t, err)

	_, err = svc.SetEnabled(ctx, f.ID, service.SetEnabledInput{Enabled: boolPtr(false)})
	require.NoError(t, err)
	n1, _ := st.CountHistoriesByFlag(ctx, f.ID)

	_, err = svc.SetEnabled(ctx, f.ID, service.SetEnabledInput{Enabled: boolPtr(false)})
	require.NoError(t, err)
	n2, _ := st.CountHistoriesByFlag(ctx, f.ID)
	require.Equal(t, n1, n2)
}

func TestEvaluateSeedRuleMatch(t *testing.T) {
	svc, _ := setupService(t)
	ctx := context.Background()
	out, err := svc.Evaluate(ctx, service.EvaluateInput{
		Key: "checkout_v2", Environment: "development", Attributes: map[string]any{"country": "CN"},
	})
	require.NoError(t, err)
	require.True(t, out.FinalValue)
	require.True(t, out.Matched)
	require.Equal(t, model.ReasonRuleMatched, out.Reason)

	out2, err := svc.Evaluate(ctx, service.EvaluateInput{
		Key: "dark_mode", Environment: "development", Attributes: map[string]any{},
	})
	require.NoError(t, err)
	require.True(t, out2.FinalValue)
	require.Equal(t, model.ReasonDefaultValue, out2.Reason)
	_ = fmt.Sprintf("%v", time.Now())
}
