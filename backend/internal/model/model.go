package model

import "time"

const ActorLocalAdmin = "local-admin"

type Flag struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Key          string    `db:"key" json:"key"`
	Environment  string    `db:"environment" json:"environment"`
	Enabled      bool      `db:"enabled" json:"enabled"`
	DefaultValue bool      `db:"default_value" json:"defaultValue"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

type Rule struct {
	ID            int64     `db:"id" json:"id"`
	FlagID        int64     `db:"flag_id" json:"flagId"`
	Attribute     string    `db:"attribute" json:"attribute"`
	Operator      string    `db:"operator" json:"operator"`
	ExpectedValue string    `db:"expected_value" json:"expectedValue"`
	ReturnValue   bool      `db:"return_value" json:"returnValue"`
	Priority      int       `db:"priority" json:"priority"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

type History struct {
	ID        int64     `db:"id" json:"id"`
	FlagID    int64     `db:"flag_id" json:"flagId"`
	Actor     string    `db:"actor" json:"actor"`
	Action    string    `db:"action" json:"action"`
	Summary   string    `db:"summary" json:"summary"`
	Payload   *string   `db:"payload" json:"payload,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

const (
	ActionCreateFlag  = "CREATE_FLAG"
	ActionUpdateFlag  = "UPDATE_FLAG"
	ActionEnableFlag  = "ENABLE_FLAG"
	ActionDisableFlag = "DISABLE_FLAG"
	ActionCreateRule  = "CREATE_RULE"
	ActionUpdateRule  = "UPDATE_RULE"
	ActionDeleteRule  = "DELETE_RULE"
)

const (
	ReasonFlagDisabled = "FLAG_DISABLED"
	ReasonRuleMatched  = "RULE_MATCHED"
	ReasonDefaultValue = "DEFAULT_VALUE"
)
