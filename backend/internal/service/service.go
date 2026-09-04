package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"featureflag/internal/eval"
	"featureflag/internal/model"
	"featureflag/internal/store"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidJSON       = errors.New("invalid json")
	ErrDuplicatePriority = errors.New("duplicate priority")
	ErrFlagNotFound      = errors.New("flag not found")
	ErrRuleNotFound      = errors.New("rule not found")
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,63}$`)

type Service struct {
	Store *store.Store
	// FailHistoryInsert is test-only: when true, history insert fails to verify rollback.
	FailHistoryInsert bool
}

func New(st *store.Store) *Service {
	return &Service{Store: st}
}

type CreateFlagInput struct {
	Name         string `json:"name"`
	Key          string `json:"key"`
	Environment  string `json:"environment"`
	Enabled      *bool  `json:"enabled"`
	DefaultValue *bool  `json:"defaultValue"`
}

type UpdateFlagInput struct {
	Name         string `json:"name"`
	DefaultValue *bool  `json:"defaultValue"`
}

type SetEnabledInput struct {
	Enabled *bool `json:"enabled"`
}

type RuleInput struct {
	Attribute     string `json:"attribute"`
	Operator      string `json:"operator"`
	ExpectedValue any    `json:"expectedValue"`
	ReturnValue   *bool  `json:"returnValue"`
	Priority      *int   `json:"priority"`
}

type EvaluateInput struct {
	Key         string         `json:"key"`
	Environment string         `json:"environment"`
	Attributes  map[string]any `json:"attributes"`
}

type EvaluateOutput struct {
	FinalValue  bool        `json:"finalValue"`
	Matched     bool        `json:"matched"`
	MatchedRule *model.Rule `json:"matchedRule"`
	Reason      string      `json:"reason"`
	Flag        model.Flag  `json:"flag"`
}

type FlagDetail struct {
	Flag       model.Flag      `json:"flag"`
	Rules      []model.Rule    `json:"rules"`
	Histories  []model.History `json:"histories"`
}

func validEnv(e string) bool {
	switch e {
	case "development", "staging", "production":
		return true
	default:
		return false
	}
}

func (s *Service) ListFlags(ctx context.Context, q, env string, enabled *bool) ([]model.Flag, error) {
	if env != "" && !validEnv(env) {
		return nil, fmt.Errorf("%w: invalid environment", ErrInvalidInput)
	}
	return s.Store.ListFlags(ctx, q, env, enabled)
}

func (s *Service) GetDetail(ctx context.Context, id int64) (*FlagDetail, error) {
	f, err := s.Store.GetFlag(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	}
	if err != nil {
		return nil, err
	}
	rules, err := s.Store.ListRules(ctx, id)
	if err != nil {
		return nil, err
	}
	histories, err := s.Store.ListHistories(ctx, id)
	if err != nil {
		return nil, err
	}
	return &FlagDetail{Flag: *f, Rules: rules, Histories: histories}, nil
}

func (s *Service) CreateFlag(ctx context.Context, in CreateFlagInput) (*model.Flag, error) {
	name := strings.TrimSpace(in.Name)
	key := strings.TrimSpace(in.Key)
	if name == "" || len(name) > 100 {
		return nil, fmt.Errorf("%w: name required and <=100", ErrInvalidInput)
	}
	if !keyPattern.MatchString(key) {
		return nil, fmt.Errorf("%w: invalid key format", ErrInvalidInput)
	}
	if !validEnv(in.Environment) {
		return nil, fmt.Errorf("%w: invalid environment", ErrInvalidInput)
	}
	if in.DefaultValue == nil {
		return nil, fmt.Errorf("%w: defaultValue required", ErrInvalidInput)
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	f := &model.Flag{
		Name: name, Key: key, Environment: in.Environment,
		Enabled: enabled, DefaultValue: *in.DefaultValue,
	}
	if err := s.Store.InsertFlagTx(ctx, tx, f); err != nil {
		if errors.Is(err, store.ErrFlagKeyConflict) {
			return nil, err
		}
		return nil, err
	}

	h := &model.History{
		FlagID:  f.ID,
		Actor:   model.ActorLocalAdmin,
		Action:  model.ActionCreateFlag,
		Summary: fmt.Sprintf("create flag name=%s key=%s env=%s enabled=%v default=%v", f.Name, f.Key, f.Environment, f.Enabled, f.DefaultValue),
	}
	if s.FailHistoryInsert {
		return nil, errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return f, nil
}

func (s *Service) UpdateFlag(ctx context.Context, id int64, in UpdateFlagInput) (*model.Flag, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 100 {
		return nil, fmt.Errorf("%w: name required and <=100", ErrInvalidInput)
	}
	if in.DefaultValue == nil {
		return nil, fmt.Errorf("%w: defaultValue required", ErrInvalidInput)
	}
	old, err := s.Store.GetFlag(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	}
	if err != nil {
		return nil, err
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	f, err := s.Store.UpdateFlagTx(ctx, tx, id, name, *in.DefaultValue)
	if err != nil {
		return nil, err
	}
	h := &model.History{
		FlagID: id,
		Actor:  model.ActorLocalAdmin,
		Action: model.ActionUpdateFlag,
		Summary: fmt.Sprintf("name: %s → %s; default_value: %v → %v",
			old.Name, f.Name, old.DefaultValue, f.DefaultValue),
	}
	if s.FailHistoryInsert {
		return nil, errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return f, nil
}

func (s *Service) SetEnabled(ctx context.Context, id int64, in SetEnabledInput) (*model.Flag, error) {
	if in.Enabled == nil {
		return nil, fmt.Errorf("%w: enabled required", ErrInvalidInput)
	}
	old, err := s.Store.GetFlag(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	}
	if err != nil {
		return nil, err
	}
	if old.Enabled == *in.Enabled {
		return old, nil
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	f, err := s.Store.SetEnabledTx(ctx, tx, id, *in.Enabled)
	if err != nil {
		return nil, err
	}
	action := model.ActionEnableFlag
	if !*in.Enabled {
		action = model.ActionDisableFlag
	}
	h := &model.History{
		FlagID:  id,
		Actor:   model.ActorLocalAdmin,
		Action:  action,
		Summary: fmt.Sprintf("enabled: %v → %v", old.Enabled, f.Enabled),
	}
	if s.FailHistoryInsert {
		return nil, errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return f, nil
}

func normalizeExpected(operator string, raw any) (string, error) {
	switch operator {
	case "equals":
		str, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("%w: equals expectedValue must be string", ErrInvalidInput)
		}
		str = strings.TrimSpace(str)
		if str == "" {
			return "", fmt.Errorf("%w: equals expectedValue required", ErrInvalidInput)
		}
		return str, nil
	case "in":
		switch v := raw.(type) {
		case []any:
			items := make([]string, 0, len(v))
			for _, x := range v {
				s, ok := x.(string)
				if !ok {
					return "", fmt.Errorf("%w: in expectedValue must be string array", ErrInvalidInput)
				}
				items = append(items, strings.TrimSpace(s))
			}
			b, err := json.Marshal(items)
			if err != nil {
				return "", err
			}
			return string(b), nil
		case []string:
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		default:
			return "", fmt.Errorf("%w: in expectedValue must be string array", ErrInvalidInput)
		}
	default:
		return "", fmt.Errorf("%w: operator must be equals or in", ErrInvalidInput)
	}
}

func validateRuleInput(in RuleInput) (attribute, operator, expected string, returnValue bool, priority int, err error) {
	attribute = strings.TrimSpace(in.Attribute)
	if attribute == "" {
		err = fmt.Errorf("%w: attribute required", ErrInvalidInput)
		return
	}
	operator = strings.TrimSpace(in.Operator)
	if operator != "equals" && operator != "in" {
		err = fmt.Errorf("%w: operator must be equals or in", ErrInvalidInput)
		return
	}
	expected, err = normalizeExpected(operator, in.ExpectedValue)
	if err != nil {
		return
	}
	if in.ReturnValue == nil {
		err = fmt.Errorf("%w: returnValue required", ErrInvalidInput)
		return
	}
	returnValue = *in.ReturnValue
	if in.Priority == nil || *in.Priority < 0 {
		err = fmt.Errorf("%w: priority must be >= 0", ErrInvalidInput)
		return
	}
	priority = *in.Priority
	return
}

func (s *Service) CreateRule(ctx context.Context, flagID int64, in RuleInput) (*model.Rule, error) {
	attr, op, expected, ret, priority, err := validateRuleInput(in)
	if err != nil {
		return nil, err
	}
	if _, err := s.Store.GetFlag(ctx, flagID); errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	} else if err != nil {
		return nil, err
	}
	exists, err := s.Store.PriorityExists(ctx, flagID, priority, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicatePriority
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	r := &model.Rule{
		FlagID: flagID, Attribute: attr, Operator: op,
		ExpectedValue: expected, ReturnValue: ret, Priority: priority,
	}
	if err := s.Store.InsertRuleTx(ctx, tx, r); err != nil {
		return nil, err
	}
	h := &model.History{
		FlagID: flagID,
		Actor:  model.ActorLocalAdmin,
		Action: model.ActionCreateRule,
		Summary: fmt.Sprintf("create rule id=%d attr=%s op=%s expected=%s return=%v priority=%d",
			r.ID, r.Attribute, r.Operator, r.ExpectedValue, r.ReturnValue, r.Priority),
	}
	if s.FailHistoryInsert {
		return nil, errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return r, nil
}

func (s *Service) UpdateRule(ctx context.Context, flagID, ruleID int64, in RuleInput) (*model.Rule, error) {
	attr, op, expected, ret, priority, err := validateRuleInput(in)
	if err != nil {
		return nil, err
	}
	old, err := s.Store.GetRule(ctx, flagID, ruleID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	exists, err := s.Store.PriorityExists(ctx, flagID, priority, ruleID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicatePriority
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	r := &model.Rule{
		ID: ruleID, FlagID: flagID, Attribute: attr, Operator: op,
		ExpectedValue: expected, ReturnValue: ret, Priority: priority,
	}
	if err := s.Store.UpdateRuleTx(ctx, tx, r); err != nil {
		return nil, err
	}
	h := &model.History{
		FlagID: flagID,
		Actor:  model.ActorLocalAdmin,
		Action: model.ActionUpdateRule,
		Summary: fmt.Sprintf("update rule id=%d: attr %s→%s; op %s→%s; expected %s→%s; return %v→%v; priority %d→%d",
			ruleID, old.Attribute, r.Attribute, old.Operator, r.Operator, old.ExpectedValue, r.ExpectedValue,
			old.ReturnValue, r.ReturnValue, old.Priority, r.Priority),
	}
	if s.FailHistoryInsert {
		return nil, errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return r, nil
}

func (s *Service) DeleteRule(ctx context.Context, flagID, ruleID int64) error {
	old, err := s.Store.GetRule(ctx, flagID, ruleID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrRuleNotFound
	}
	if err != nil {
		return err
	}

	tx, err := s.Store.BeginTxx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := s.Store.DeleteRuleTx(ctx, tx, flagID, ruleID); err != nil {
		return err
	}
	h := &model.History{
		FlagID: flagID,
		Actor:  model.ActorLocalAdmin,
		Action: model.ActionDeleteRule,
		Summary: fmt.Sprintf("delete rule id=%d attr=%s op=%s expected=%s return=%v priority=%d",
			old.ID, old.Attribute, old.Operator, old.ExpectedValue, old.ReturnValue, old.Priority),
	}
	if s.FailHistoryInsert {
		return errors.New("forced history insert failure")
	}
	if err := s.Store.InsertHistoryTx(ctx, tx, h); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *Service) Evaluate(ctx context.Context, in EvaluateInput) (*EvaluateOutput, error) {
	key := strings.TrimSpace(in.Key)
	env := strings.TrimSpace(in.Environment)
	if key == "" || !validEnv(env) {
		return nil, fmt.Errorf("%w: key and valid environment required", ErrInvalidInput)
	}
	if in.Attributes == nil {
		return nil, ErrInvalidJSON
	}
	f, err := s.Store.GetFlagByKeyEnv(ctx, key, env)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	}
	if err != nil {
		return nil, err
	}
	var rules []model.Rule
	if f.Enabled {
		rules, err = s.Store.ListRules(ctx, f.ID)
		if err != nil {
			return nil, err
		}
	}
	res := eval.Evaluate(*f, rules, in.Attributes)
	return &EvaluateOutput{
		FinalValue:  res.FinalValue,
		Matched:     res.Matched,
		MatchedRule: res.MatchedRule,
		Reason:      res.Reason,
		Flag:        *f,
	}, nil
}
