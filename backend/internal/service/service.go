package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"featureflag/internal/db"
	"featureflag/internal/eval"
	"featureflag/internal/model"
	"featureflag/internal/store"

	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrFlagNotFound = errors.New("flag not found")
)

type Service struct {
	DB *db.DB
}

func New(database *db.DB) *Service {
	return &Service{DB: database}
}

func mapStoreErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func ValidEnvironment(env string) bool {
	switch env {
	case "development", "staging", "production":
		return true
	default:
		return false
	}
}

func ValidOperator(op string) bool {
	return op == "equals" || op == "in"
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
	Version      *int64 `json:"version"` // 客户端看到的 version，乐观锁用；不可用服务端最新读替代
}

type RuleInput struct {
	Attribute     string `json:"attribute"`
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expectedValue"`
	ReturnValue   *bool  `json:"returnValue"`
	Priority      *int   `json:"priority"`
	Version       *int64 `json:"version"` // 仅 UpdateRule 必填；CreateRule 忽略
}

type FlagDetail struct {
	Flag  model.Flag   `json:"flag"`
	Rules []model.Rule `json:"rules"`
}

func (s *Service) ListFlags(ctx context.Context, f store.FlagFilter) ([]model.Flag, error) {
	if f.Environment != "" && !ValidEnvironment(f.Environment) {
		return nil, fmt.Errorf("%w: environment 必须是 development/staging/production", ErrInvalidInput)
	}
	return store.ListFlags(ctx, s.DB.SQL, f)
}

func (s *Service) GetFlagDetail(ctx context.Context, id int64) (*FlagDetail, error) {
	f, err := store.GetFlag(ctx, s.DB.SQL, id)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	rules, err := store.ListRules(ctx, s.DB.SQL, id)
	if err != nil {
		return nil, err
	}
	return &FlagDetail{Flag: *f, Rules: rules}, nil
}

func (s *Service) ListHistory(ctx context.Context, flagID int64) ([]model.History, error) {
	if _, err := store.GetFlag(ctx, s.DB.SQL, flagID); err != nil {
		return nil, mapStoreErr(err)
	}
	return store.ListHistory(ctx, s.DB.SQL, flagID)
}

func (s *Service) CreateFlag(ctx context.Context, in CreateFlagInput) (*model.Flag, error) {
	name := strings.TrimSpace(in.Name)
	key := strings.TrimSpace(in.Key)
	if name == "" {
		return nil, fmt.Errorf("%w: name 必填", ErrInvalidInput)
	}
	if key == "" {
		return nil, fmt.Errorf("%w: key 必填", ErrInvalidInput)
	}
	if !ValidEnvironment(in.Environment) {
		return nil, fmt.Errorf("%w: environment 必须是 development/staging/production", ErrInvalidInput)
	}
	if in.DefaultValue == nil {
		return nil, fmt.Errorf("%w: defaultValue 必填", ErrInvalidInput)
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	var created model.Flag
	err := s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		created = model.Flag{
			Name:         name,
			Key:          key,
			Environment:  in.Environment,
			Enabled:      enabled,
			DefaultValue: *in.DefaultValue,
		}
		if err := store.CreateFlag(ctx, tx, &created); err != nil {
			return err
		}
		fid := created.ID
		h := &model.History{
			FlagID:        &fid,
			OperationType: model.OpCreateFlag,
			Operator:      model.ActorLocalAdmin,
			Summary: fmt.Sprintf("create flag name=%s key=%s env=%s enabled=%v defaultValue=%v",
				created.Name, created.Key, created.Environment, created.Enabled, created.DefaultValue),
		}
		return store.InsertHistory(ctx, tx, h)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *Service) UpdateFlag(ctx context.Context, id int64, in UpdateFlagInput) (*model.Flag, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name 必填", ErrInvalidInput)
	}
	if in.DefaultValue == nil {
		return nil, fmt.Errorf("%w: defaultValue 必填", ErrInvalidInput)
	}
	if in.Version == nil {
		return nil, fmt.Errorf("%w: version 必填", ErrInvalidInput)
	}
	old, err := store.GetFlag(ctx, s.DB.SQL, id)
	if err != nil {
		return nil, mapStoreErr(err)
	}

	var updated *model.Flag
	err = s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		var txErr error
		// expectedVersion 必须来自请求体 in.Version，禁止传 old.Version。
		updated, txErr = store.UpdateFlag(ctx, tx, id, name, *in.DefaultValue, *in.Version)
		if txErr != nil {
			return mapStoreErr(txErr)
		}
		fid := id
		h := &model.History{
			FlagID:        &fid,
			OperationType: model.OpUpdateFlag,
			Operator:      model.ActorLocalAdmin,
			Summary: fmt.Sprintf("name: %s → %s; default_value: %v → %v",
				old.Name, updated.Name, old.DefaultValue, updated.DefaultValue),
		}
		return store.InsertHistory(ctx, tx, h)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) setEnabled(ctx context.Context, id int64, enabled bool) (*model.Flag, error) {
	old, err := store.GetFlag(ctx, s.DB.SQL, id)
	if err != nil {
		return nil, mapStoreErr(err)
	}

	var updated *model.Flag
	err = s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		var txErr error
		updated, txErr = store.SetFlagEnabled(ctx, tx, id, enabled)
		if txErr != nil {
			return mapStoreErr(txErr)
		}
		op := model.OpEnableFlag
		if !enabled {
			op = model.OpDisableFlag
		}
		fid := id
		h := &model.History{
			FlagID:        &fid,
			OperationType: op,
			Operator:      model.ActorLocalAdmin,
			Summary:       fmt.Sprintf("enabled: %v → %v", old.Enabled, updated.Enabled),
		}
		return store.InsertHistory(ctx, tx, h)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) EnableFlag(ctx context.Context, id int64) (*model.Flag, error) {
	return s.setEnabled(ctx, id, true)
}

func (s *Service) DisableFlag(ctx context.Context, id int64) (*model.Flag, error) {
	return s.setEnabled(ctx, id, false)
}

func parseRuleInput(in RuleInput) (attribute, operator, expected string, returnValue bool, priority int, err error) {
	attribute = strings.TrimSpace(in.Attribute)
	if attribute == "" {
		err = fmt.Errorf("%w: attribute 必填", ErrInvalidInput)
		return
	}
	operator = strings.TrimSpace(in.Operator)
	if !ValidOperator(operator) {
		err = fmt.Errorf("%w: operator 必须是 equals 或 in", ErrInvalidInput)
		return
	}
	expected = strings.TrimSpace(in.ExpectedValue)
	if expected == "" {
		err = fmt.Errorf("%w: expectedValue 必填", ErrInvalidInput)
		return
	}
	if in.ReturnValue == nil {
		err = fmt.Errorf("%w: returnValue 必填", ErrInvalidInput)
		return
	}
	returnValue = *in.ReturnValue
	if in.Priority == nil {
		err = fmt.Errorf("%w: priority 必填", ErrInvalidInput)
		return
	}
	if *in.Priority < 0 {
		err = fmt.Errorf("%w: priority 必须 >= 0", ErrInvalidInput)
		return
	}
	priority = *in.Priority
	return
}

func (s *Service) CreateRule(ctx context.Context, flagID int64, in RuleInput) (*model.Rule, error) {
	attr, op, expected, ret, priority, err := parseRuleInput(in)
	if err != nil {
		return nil, err
	}
	if _, err := store.GetFlag(ctx, s.DB.SQL, flagID); err != nil {
		return nil, mapStoreErr(err)
	}

	var created model.Rule
	err = s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		exists, e := store.PriorityExists(ctx, tx, flagID, priority, 0)
		if e != nil {
			return e
		}
		if exists {
			return db.ErrRulePriorityConflict
		}
		created = model.Rule{
			FlagID:        flagID,
			Attribute:     attr,
			Operator:      op,
			ExpectedValue: expected,
			ReturnValue:   ret,
			Priority:      priority,
		}
		if e := store.CreateRule(ctx, tx, &created); e != nil {
			return e
		}
		fid := flagID
		h := &model.History{
			FlagID:        &fid,
			OperationType: model.OpCreateRule,
			Operator:      model.ActorLocalAdmin,
			Summary: fmt.Sprintf("create rule id=%d attr=%s op=%s expected=%s return=%v priority=%d",
				created.ID, created.Attribute, created.Operator, created.ExpectedValue, created.ReturnValue, created.Priority),
		}
		return store.InsertHistory(ctx, tx, h)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *Service) UpdateRule(ctx context.Context, flagID, ruleID int64, in RuleInput) (*model.Rule, error) {
	attr, op, expected, ret, priority, err := parseRuleInput(in)
	if err != nil {
		return nil, err
	}
	if in.Version == nil {
		return nil, fmt.Errorf("%w: version 必填", ErrInvalidInput)
	}
	old, err := store.GetRule(ctx, s.DB.SQL, flagID, ruleID)
	if err != nil {
		return nil, mapStoreErr(err)
	}

	var updated model.Rule
	err = s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		exists, e := store.PriorityExists(ctx, tx, flagID, priority, ruleID)
		if e != nil {
			return e
		}
		if exists {
			return db.ErrRulePriorityConflict
		}
		updated = model.Rule{
			ID:            ruleID,
			FlagID:        flagID,
			Attribute:     attr,
			Operator:      op,
			ExpectedValue: expected,
			ReturnValue:   ret,
			Priority:      priority,
			Version:       *in.Version, // 客户端快照，供 store WHERE version=? 使用
		}
		if e := store.UpdateRule(ctx, tx, &updated); e != nil {
			return mapStoreErr(e)
		}
		fid := flagID
		h := &model.History{
			FlagID:        &fid,
			OperationType: model.OpUpdateRule,
			Operator:      model.ActorLocalAdmin,
			Summary: fmt.Sprintf("update rule id=%d: attr %s→%s; op %s→%s; expected %s→%s; return %v→%v; priority %d→%d",
				ruleID, old.Attribute, updated.Attribute, old.Operator, updated.Operator,
				old.ExpectedValue, updated.ExpectedValue, old.ReturnValue, updated.ReturnValue,
				old.Priority, updated.Priority),
		}
		return store.InsertHistory(ctx, tx, h)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Service) DeleteRule(ctx context.Context, flagID, ruleID int64) error {
	old, err := store.GetRule(ctx, s.DB.SQL, flagID, ruleID)
	if err != nil {
		return mapStoreErr(err)
	}
	return s.DB.WithTx(ctx, func(tx *sqlx.Tx) error {
		if e := store.DeleteRule(ctx, tx, flagID, ruleID); e != nil {
			return mapStoreErr(e)
		}
		fid := flagID
		h := &model.History{
			FlagID:        &fid,
			OperationType: model.OpDeleteRule,
			Operator:      model.ActorLocalAdmin,
			Summary: fmt.Sprintf("delete rule id=%d attr=%s op=%s expected=%s return=%v priority=%d",
				old.ID, old.Attribute, old.Operator, old.ExpectedValue, old.ReturnValue, old.Priority),
		}
		return store.InsertHistory(ctx, tx, h)
	})
}

func (s *Service) Evaluate(ctx context.Context, key, env string, attrs map[string]any) (*eval.Result, error) {
	key = strings.TrimSpace(key)
	env = strings.TrimSpace(env)
	if key == "" {
		return nil, fmt.Errorf("%w: key 必填", ErrInvalidInput)
	}
	if !ValidEnvironment(env) {
		return nil, fmt.Errorf("%w: environment 必须是 development/staging/production", ErrInvalidInput)
	}
	f, err := store.GetFlagByKeyAndEnv(ctx, s.DB.SQL, key, env)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrFlagNotFound
	}
	if err != nil {
		return nil, err
	}
	rules, err := store.ListRules(ctx, s.DB.SQL, f.ID)
	if err != nil {
		return nil, err
	}
	res := eval.Evaluate(*f, rules, attrs)
	return &res, nil
}
