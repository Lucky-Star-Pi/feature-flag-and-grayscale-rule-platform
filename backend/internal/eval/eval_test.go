package eval

import (
	"testing"

	"featureflag/internal/model"

	"github.com/stretchr/testify/require"
)

func baseFlag(enabled bool, def bool) model.Flag {
	return model.Flag{ID: 1, Key: "f", Environment: "development", Enabled: enabled, DefaultValue: def}
}

func TestDisabledShortCircuit(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, FlagID: 1, Attribute: "country", Operator: "equals",
		ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	res := Evaluate(baseFlag(false, true), rules, map[string]any{"country": "CN"})
	require.False(t, res.FinalValue)
	require.False(t, res.Matched)
	require.Nil(t, res.MatchedRule)
	require.Equal(t, model.ReasonFlagDisabled, res.Reason)
}

func TestPriorityShortCircuit(t *testing.T) {
	rules := []model.Rule{
		{ID: 2, FlagID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: false, Priority: 10},
		{ID: 1, FlagID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0},
	}
	res := Evaluate(baseFlag(true, false), rules, map[string]any{"country": "CN"})
	require.True(t, res.FinalValue)
	require.True(t, res.Matched)
	require.NotNil(t, res.MatchedRule)
	require.Equal(t, 0, res.MatchedRule.Priority)
	require.Equal(t, model.ReasonRuleMatched, res.Reason)
}

func TestDefaultValueTrueAndFalse(t *testing.T) {
	t.Run("default true", func(t *testing.T) {
		res := Evaluate(baseFlag(true, true), nil, map[string]any{"x": "y"})
		require.True(t, res.FinalValue)
		require.False(t, res.Matched)
		require.Equal(t, model.ReasonDefaultValue, res.Reason)
	})
	t.Run("default false", func(t *testing.T) {
		rules := []model.Rule{{
			ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "US", ReturnValue: true, Priority: 0,
		}}
		res := Evaluate(baseFlag(true, false), rules, map[string]any{"country": "CN"})
		require.False(t, res.FinalValue)
		require.False(t, res.Matched)
		require.Equal(t, model.ReasonDefaultValue, res.Reason)
	})
}

func TestEqualsHitAndMiss(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	hit := Evaluate(baseFlag(true, false), rules, map[string]any{"country": "CN"})
	require.True(t, hit.Matched)
	require.True(t, hit.FinalValue)

	miss := Evaluate(baseFlag(true, false), rules, map[string]any{"country": "US"})
	require.False(t, miss.Matched)
	require.False(t, miss.FinalValue)
}

func TestInHitMissEmpty(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "plan", Operator: "in", ExpectedValue: `["pro","enterprise"]`, ReturnValue: true, Priority: 0,
	}}
	hit := Evaluate(baseFlag(true, false), rules, map[string]any{"plan": "pro"})
	require.True(t, hit.Matched)

	miss := Evaluate(baseFlag(true, false), rules, map[string]any{"plan": "free"})
	require.False(t, miss.Matched)

	empty := []model.Rule{{
		ID: 2, Attribute: "plan", Operator: "in", ExpectedValue: `[]`, ReturnValue: true, Priority: 0,
	}}
	res := Evaluate(baseFlag(true, false), empty, map[string]any{"plan": "pro"})
	require.False(t, res.Matched)
}

func TestAttributeMissingAndTypes(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	missing := Evaluate(baseFlag(true, false), rules, map[string]any{"plan": "pro"})
	require.False(t, missing.Matched)

	nullAttr := Evaluate(baseFlag(true, false), rules, map[string]any{"country": nil})
	require.False(t, nullAttr.Matched)

	num := []model.Rule{{
		ID: 1, Attribute: "user_id", Operator: "equals", ExpectedValue: "1", ReturnValue: true, Priority: 0,
	}}
	numHit := Evaluate(baseFlag(true, false), num, map[string]any{"user_id": float64(1)})
	require.True(t, numHit.Matched)

	boolHit := Evaluate(baseFlag(true, false), []model.Rule{{
		ID: 1, Attribute: "vip", Operator: "equals", ExpectedValue: "true", ReturnValue: true, Priority: 0,
	}}, map[string]any{"vip": true})
	require.True(t, boolHit.Matched)

	objMiss := Evaluate(baseFlag(true, false), rules, map[string]any{"country": map[string]any{"a": 1}})
	require.False(t, objMiss.Matched)

	arrMiss := Evaluate(baseFlag(true, false), rules, map[string]any{"country": []any{"CN"}})
	require.False(t, arrMiss.Matched)
}

func TestTrimAndCaseSensitive(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	trimHit := Evaluate(baseFlag(true, false), rules, map[string]any{"country": " CN "})
	require.True(t, trimHit.Matched)

	caseMiss := Evaluate(baseFlag(true, false), rules, map[string]any{"country": "cn"})
	require.False(t, caseMiss.Matched)
}

func TestUnsortedInputStillPriorityOrder(t *testing.T) {
	rules := []model.Rule{
		{ID: 10, Attribute: "x", Operator: "equals", ExpectedValue: "1", ReturnValue: false, Priority: 5},
		{ID: 3, Attribute: "x", Operator: "equals", ExpectedValue: "1", ReturnValue: true, Priority: 1},
	}
	res := Evaluate(baseFlag(true, false), rules, map[string]any{"x": "1"})
	require.True(t, res.FinalValue)
	require.Equal(t, 1, res.MatchedRule.Priority)
}
