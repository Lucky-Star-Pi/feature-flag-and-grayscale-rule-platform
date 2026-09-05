-- demo 种子：2 个 Flag + 若干规则（同 Key 不同环境验证 UNIQUE(key, environment)）

INSERT INTO flags (name, key, environment, enabled, default_value)
VALUES
    ('Checkout V2', 'checkout_v2', 'development', TRUE, FALSE),
    ('Checkout V2', 'checkout_v2', 'production', FALSE, FALSE);

INSERT INTO rules (flag_id, attribute, operator, expected_value, return_value, priority)
SELECT id, 'country', 'equals', 'CN', TRUE, 0
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO rules (flag_id, attribute, operator, expected_value, return_value, priority)
SELECT id, 'plan', 'in', '["pro","enterprise"]', TRUE, 10
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO history (flag_id, operation_type, operator, summary)
SELECT id, 'CREATE_FLAG', 'local-admin', 'seed: create checkout_v2@development'
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO history (flag_id, operation_type, operator, summary)
SELECT id, 'CREATE_FLAG', 'local-admin', 'seed: create checkout_v2@production (disabled)'
FROM flags WHERE key = 'checkout_v2' AND environment = 'production';
