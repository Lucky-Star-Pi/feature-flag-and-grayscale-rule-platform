package eval

import (
	"encoding/json"
	"sort"
	"strconv"

	"featureflag/internal/model"
)

// 评估语义（与 README / 前端文案三处一致）：
//  1. 停用短路：enabled=false → value=false, matched=false, reason="disabled"（不看任何规则）
//  2. 有序匹配：priority ASC，同分 id ASC；第一条命中即停，reason="matched"
//  3. 默认值兜底：无命中 → value=flag.default_value, matched=false, reason="default"
//  4. equals：Stringify(属性值) 与 expected_value 精确相等
//  5. in：expected_value 解析为 JSON 字符串数组，成员精确包含；解析失败或空数组 → 本规则跳过
//  6. 属性缺失 / 值为 null、对象、数组 → 本规则跳过

const (
	ReasonDisabled = "disabled"
	ReasonMatched  = "matched"
	ReasonDefault  = "default"
)

type Result struct {
	Value       bool        `json:"value"`
	Matched     bool        `json:"matched"`
	MatchedRule *model.Rule `json:"matchedRule"`
	Reason      string      `json:"reason"`
}

func Evaluate(flag model.Flag, rules []model.Rule, attrs map[string]any) Result {
	if !flag.Enabled {
		return Result{Value: false, Matched: false, MatchedRule: nil, Reason: ReasonDisabled}
	}

	sorted := make([]model.Rule, len(rules))
	copy(sorted, rules)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})

	if attrs == nil {
		attrs = map[string]any{}
	}

	for i := range sorted {
		r := sorted[i]
		if matchRule(r, attrs) {
			cp := r
			return Result{
				Value:       r.ReturnValue,
				Matched:     true,
				MatchedRule: &cp,
				Reason:      ReasonMatched,
			}
		}
	}

	return Result{
		Value:       flag.DefaultValue,
		Matched:     false,
		MatchedRule: nil,
		Reason:      ReasonDefault,
	}
}

func matchRule(r model.Rule, attrs map[string]any) bool {
	raw, ok := attrs[r.Attribute]
	if !ok {
		return false
	}
	got, ok := Stringify(raw)
	if !ok {
		return false
	}
	switch r.Operator {
	case "equals":
		return got == r.ExpectedValue
	case "in":
		items, ok := ParseInValues(r.ExpectedValue)
		if !ok || len(items) == 0 {
			return false
		}
		for _, item := range items {
			if got == item {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Stringify 按类型归一：string 原样；JSON number(float64) 用 FormatFloat；
// bool → "true"/"false"；null / 对象 / 数组 → ( "", false ) 视为缺失。
func Stringify(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 64), true
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return t.String(), true
		}
		return strconv.FormatFloat(f, 'f', -1, 64), true
	case int:
		return strconv.FormatInt(int64(t), 10), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case int32:
		return strconv.FormatInt(int64(t), 10), true
	default:
		return "", false
	}
}

// ParseInValues 将 expected_value 解析为 JSON 字符串数组。
// 非数组、非全 string、非法 JSON → false。
func ParseInValues(expected string) ([]string, bool) {
	var items []string
	if err := json.Unmarshal([]byte(expected), &items); err != nil {
		return nil, false
	}
	return items, true
}
