package eval

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"featureflag/internal/model"
)

// Result is the outcome of evaluating a flag against user attributes.
type Result struct {
	FinalValue  bool         `json:"finalValue"`
	Matched     bool         `json:"matched"`
	MatchedRule *model.Rule  `json:"matchedRule"`
	Reason      string       `json:"reason"`
}

// Evaluate applies the locked semantics:
//  1. enabled=false => false, FLAG_DISABLED (no rule evaluation)
//  2. rules sorted by priority ASC, id ASC; first hit wins
//  3. otherwise default_value with DEFAULT_VALUE
func Evaluate(flag model.Flag, rules []model.Rule, attrs map[string]any) Result {
	if !flag.Enabled {
		return Result{
			FinalValue:  false,
			Matched:     false,
			MatchedRule: nil,
			Reason:      model.ReasonFlagDisabled,
		}
	}

	sorted := make([]model.Rule, len(rules))
	copy(sorted, rules)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})

	for i := range sorted {
		r := sorted[i]
		if matchRule(r, attrs) {
			cp := r
			return Result{
				FinalValue:  r.ReturnValue,
				Matched:     true,
				MatchedRule: &cp,
				Reason:      model.ReasonRuleMatched,
			}
		}
	}

	return Result{
		FinalValue:  flag.DefaultValue,
		Matched:     false,
		MatchedRule: nil,
		Reason:      model.ReasonDefaultValue,
	}
}

func matchRule(r model.Rule, attrs map[string]any) bool {
	raw, ok := attrs[r.Attribute]
	if !ok || raw == nil {
		return false
	}
	userStr, ok := scalarToString(raw)
	if !ok {
		return false
	}
	userStr = strings.TrimSpace(userStr)

	switch r.Operator {
	case "equals":
		return userStr == strings.TrimSpace(r.ExpectedValue)
	case "in":
		var items []string
		if err := json.Unmarshal([]byte(r.ExpectedValue), &items); err != nil {
			return false
		}
		if len(items) == 0 {
			return false
		}
		for _, item := range items {
			if userStr == strings.TrimSpace(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func scalarToString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		// JSON numbers decode as float64
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t)), true
		}
		return fmt.Sprint(t), true
	case json.Number:
		return t.String(), true
	case bool:
		return fmt.Sprint(t), true
	case int:
		return fmt.Sprintf("%d", t), true
	case int64:
		return fmt.Sprintf("%d", t), true
	default:
		return "", false
	}
}
