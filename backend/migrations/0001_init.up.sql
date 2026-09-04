CREATE TABLE IF NOT EXISTS flags (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    key             TEXT NOT NULL,
    environment     TEXT NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    default_value   BOOLEAN NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT flags_environment_key_uk UNIQUE (environment, key)
);

CREATE TABLE IF NOT EXISTS flag_rules (
    id              BIGSERIAL PRIMARY KEY,
    flag_id         BIGINT NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    attribute       TEXT NOT NULL,
    operator        TEXT NOT NULL CHECK (operator IN ('equals', 'in')),
    expected_value  TEXT NOT NULL,
    return_value    BOOLEAN NOT NULL,
    priority        INT NOT NULL CHECK (priority >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT flag_rules_flag_id_priority_uk UNIQUE (flag_id, priority)
);

CREATE INDEX IF NOT EXISTS idx_flag_rules_flag_priority
    ON flag_rules (flag_id, priority ASC, id ASC);

CREATE TABLE IF NOT EXISTS operation_histories (
    id          BIGSERIAL PRIMARY KEY,
    flag_id     BIGINT NOT NULL REFERENCES flags(id) ON DELETE RESTRICT,
    actor       TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN (
        'CREATE_FLAG', 'UPDATE_FLAG', 'ENABLE_FLAG', 'DISABLE_FLAG',
        'CREATE_RULE', 'UPDATE_RULE', 'DELETE_RULE'
    )),
    summary     TEXT NOT NULL,
    payload     JSONB NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operation_histories_flag_created
    ON operation_histories (flag_id, created_at DESC);
