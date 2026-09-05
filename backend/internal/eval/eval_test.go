package eval

import (
	"testing"

	"featureflag/internal/model"

	"github.com/stretchr/testify/require"
)

func flag(enabled, def bool) model.Flag {
	return model.Flag{ID: 1, Key: "f", Environment: "development", Enabled: enabled, DefaultValue: def}
}

func TestEvaluate_DisabledShortCircuit(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals",
		ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	res := Evaluate(flag(false, true), rules, map[string]any{"country": "CN"})
	require.False(t, res.Value)
	require.False(t, res.Matched)
	require.Nil(t, res.MatchedRule)
	require.Equal(t, ReasonDisabled, res.Reason)
}

func TestEvaluate_PriorityShortCircuit(t *testing.T) {
	rules := []model.Rule{
		{ID: 2, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: false, Priority: 10},
		{ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0},
	}
	res := Evaluate(flag(true, false), rules, map[string]any{"country": "CN"})
	require.True(t, res.Value)
	require.True(t, res.Matched)
	require.NotNil(t, res.MatchedRule)
	require.Equal(t, int64(1), res.MatchedRule.ID)
	require.Equal(t, 0, res.MatchedRule.Priority)
	require.Equal(t, ReasonMatched, res.Reason)
}

func TestEvaluate_DefaultTrueAndFalse(t *testing.T) {
	t.Run("default true", func(t *testing.T) {
		res := Evaluate(flag(true, true), nil, map[string]any{"x": "y"})
		require.True(t, res.Value)
		require.False(t, res.Matched)
		require.Equal(t, ReasonDefault, res.Reason)
	})
	t.Run("default false", func(t *testing.T) {
		rules := []model.Rule{{
			ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "US", ReturnValue: true, Priority: 0,
		}}
		res := Evaluate(flag(true, false), rules, map[string]any{"country": "CN"})
		require.False(t, res.Value)
		require.False(t, res.Matched)
		require.Equal(t, ReasonDefault, res.Reason)
	})
}

func TestEvaluate_EqualsHitAndMiss(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	hit := Evaluate(flag(true, false), rules, map[string]any{"country": "CN"})
	require.True(t, hit.Matched)
	require.True(t, hit.Value)

	miss := Evaluate(flag(true, false), rules, map[string]any{"country": "US"})
	require.False(t, miss.Matched)
	require.False(t, miss.Value)
	require.Equal(t, ReasonDefault, miss.Reason)
}

func TestEvaluate_InHitMissEmptyAndInvalidJSON(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "plan", Operator: "in", ExpectedValue: `["pro","enterprise"]`, ReturnValue: true, Priority: 0,
	}}
	hit := Evaluate(flag(true, false), rules, map[string]any{"plan": "pro"})
	require.True(t, hit.Matched)

	miss := Evaluate(flag(true, false), rules, map[string]any{"plan": "free"})
	require.False(t, miss.Matched)

	empty := []model.Rule{{
		ID: 2, Attribute: "plan", Operator: "in", ExpectedValue: `[]`, ReturnValue: true, Priority: 0,
	}}
	res := Evaluate(flag(true, false), empty, map[string]any{"plan": "pro"})
	require.False(t, res.Matched)
	require.Equal(t, ReasonDefault, res.Reason)

	bad := []model.Rule{{
		ID: 3, Attribute: "plan", Operator: "in", ExpectedValue: "not-json", ReturnValue: true, Priority: 0,
	}}
	skip := Evaluate(flag(true, false), bad, map[string]any{"plan": "pro"})
	require.False(t, skip.Matched)
	require.Equal(t, ReasonDefault, skip.Reason)
}

func TestEvaluate_AttributeMissing(t *testing.T) {
	rules := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	res := Evaluate(flag(true, false), rules, map[string]any{"plan": "pro"})
	require.False(t, res.Matched)
	require.Equal(t, ReasonDefault, res.Reason)
}

func TestEvaluate_TypeNormalize(t *testing.T) {
	num := []model.Rule{{
		ID: 1, Attribute: "user_id", Operator: "equals", ExpectedValue: "123", ReturnValue: true, Priority: 0,
	}}
	require.True(t, Evaluate(flag(true, false), num, map[string]any{"user_id": float64(123)}).Matched)

	b := []model.Rule{{
		ID: 1, Attribute: "vip", Operator: "equals", ExpectedValue: "true", ReturnValue: true, Priority: 0,
	}}
	require.True(t, Evaluate(flag(true, false), b, map[string]any{"vip": true}).Matched)

	eq := []model.Rule{{
		ID: 1, Attribute: "country", Operator: "equals", ExpectedValue: "CN", ReturnValue: true, Priority: 0,
	}}
	require.False(t, Evaluate(flag(true, false), eq, map[string]any{"country": nil}).Matched)
	require.False(t, Evaluate(flag(true, false), eq, map[string]any{"country": map[string]any{"a": 1}}).Matched)
	require.False(t, Evaluate(flag(true, false), eq, map[string]any{"country": []any{"CN"}}).Matched)
}

func TestEvaluate_SamePriorityByIDAsc(t *testing.T) {
	rules := []model.Rule{
		{ID: 20, Attribute: "x", Operator: "equals", ExpectedValue: "1", ReturnValue: false, Priority: 5},
		{ID: 3, Attribute: "x", Operator: "equals", ExpectedValue: "1", ReturnValue: true, Priority: 5},
	}
	res := Evaluate(flag(true, false), rules, map[string]any{"x": "1"})
	require.True(t, res.Value)
	require.Equal(t, int64(3), res.MatchedRule.ID)
}

func TestStringify(t *testing.T) {
	s, ok := Stringify("abc")
	require.True(t, ok)
	require.Equal(t, "abc", s)

	s, ok = Stringify(float64(123))
	require.True(t, ok)
	require.Equal(t, "123", s)

	s, ok = Stringify(float64(123.5))
	require.True(t, ok)
	require.Equal(t, "123.5", s)

	s, ok = Stringify(true)
	require.True(t, ok)
	require.Equal(t, "true", s)

	s, ok = Stringify(false)
	require.True(t, ok)
	require.Equal(t, "false", s)

	_, ok = Stringify(nil)
	require.False(t, ok)

	_, ok = Stringify(map[string]any{"a": 1})
	require.False(t, ok)

	_, ok = Stringify([]any{"x"})
	require.False(t, ok)
}

func TestParseInValues(t *testing.T) {
	items, ok := ParseInValues(`["pro","enterprise"]`)
	require.True(t, ok)
	require.Equal(t, []string{"pro", "enterprise"}, items)

	items, ok = ParseInValues(`[]`)
	require.True(t, ok)
	require.Empty(t, items)

	_, ok = ParseInValues("not-json")
	require.False(t, ok)

	_, ok = ParseInValues(`[1,2]`)
	require.False(t, ok)
}
