-- M1: 数据模型
-- 重复优先级「拒绝」策略：本表 UNIQUE(flag_id, priority) 在数据库层兜底；
-- 应用层（M2）创建/更新规则前亦可先校验并返回 400 PRIORITY_CONFLICT。

CREATE TABLE IF NOT EXISTS flags (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    key             TEXT NOT NULL,
    environment     TEXT NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    default_value   BOOLEAN NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT flags_key_environment_uk UNIQUE (key, environment)
);

CREATE TABLE IF NOT EXISTS rules (
    id              BIGSERIAL PRIMARY KEY,
    flag_id         BIGINT NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    attribute       TEXT NOT NULL,
    operator        TEXT NOT NULL CHECK (operator IN ('equals', 'in')),
    expected_value  TEXT NOT NULL,
    return_value    BOOLEAN NOT NULL,
    priority        INT NOT NULL CHECK (priority >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rules_flag_id_priority_uk UNIQUE (flag_id, priority)
);

CREATE INDEX IF NOT EXISTS idx_rules_flag_priority
    ON rules (flag_id, priority ASC, id ASC);

CREATE TABLE IF NOT EXISTS history (
    id              BIGSERIAL PRIMARY KEY,
    flag_id         BIGINT NULL REFERENCES flags(id) ON DELETE SET NULL,
    operation_type  TEXT NOT NULL,
    operator        TEXT NOT NULL,
    summary         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_history_flag_created
    ON history (flag_id, created_at DESC);
