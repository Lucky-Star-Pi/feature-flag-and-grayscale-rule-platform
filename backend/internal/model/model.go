package model

import "time"

// ActorLocalAdmin 本题不实现登录，写操作固定操作者。
const ActorLocalAdmin = "local-admin"

// 优先级方向（M2 写接口与 M3 评估必须一致）：
// 数字越小优先级越高，规则列表按 priority ASC, id ASC 返回。

const (
	OpCreateFlag  = "CREATE_FLAG"
	OpUpdateFlag  = "UPDATE_FLAG"
	OpEnableFlag  = "ENABLE_FLAG"
	OpDisableFlag = "DISABLE_FLAG"
	OpCreateRule  = "CREATE_RULE"
	OpUpdateRule  = "UPDATE_RULE"
	OpDeleteRule  = "DELETE_RULE"
)

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
	ID            int64     `db:"id" json:"id"`
	FlagID        *int64    `db:"flag_id" json:"flagId"`
	OperationType string    `db:"operation_type" json:"operationType"`
	Operator      string    `db:"operator" json:"operator"`
	Summary       string    `db:"summary" json:"summary"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}
