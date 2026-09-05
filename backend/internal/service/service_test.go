package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidEnvironment(t *testing.T) {
	require.True(t, ValidEnvironment("development"))
	require.True(t, ValidEnvironment("staging"))
	require.True(t, ValidEnvironment("production"))
	require.False(t, ValidEnvironment("prod"))
	require.False(t, ValidEnvironment(""))
}

func TestValidOperator(t *testing.T) {
	require.True(t, ValidOperator("equals"))
	require.True(t, ValidOperator("in"))
	require.False(t, ValidOperator("contains"))
	require.False(t, ValidOperator(""))
}

func TestParseRuleInput(t *testing.T) {
	ret := true
	p := 0
	attr, op, expected, rv, pr, err := parseRuleInput(RuleInput{
		Attribute: "country", Operator: "equals", ExpectedValue: "CN",
		ReturnValue: &ret, Priority: &p,
	})
	require.NoError(t, err)
	require.Equal(t, "country", attr)
	require.Equal(t, "equals", op)
	require.Equal(t, "CN", expected)
	require.True(t, rv)
	require.Equal(t, 0, pr)

	neg := -1
	_, _, _, _, _, err = parseRuleInput(RuleInput{
		Attribute: "country", Operator: "equals", ExpectedValue: "CN",
		ReturnValue: &ret, Priority: &neg,
	})
	require.ErrorIs(t, err, ErrInvalidInput)

	_, _, _, _, _, err = parseRuleInput(RuleInput{
		Attribute: "country", Operator: "gt", ExpectedValue: "CN",
		ReturnValue: &ret, Priority: &p,
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}
