INSERT INTO flags (name, key, environment, enabled, default_value)
VALUES
    ('Checkout V2', 'checkout_v2', 'development', TRUE, FALSE),
    ('Checkout V2', 'checkout_v2', 'production', FALSE, FALSE),
    ('Dark Mode', 'dark_mode', 'development', TRUE, TRUE);

INSERT INTO flag_rules (flag_id, attribute, operator, expected_value, return_value, priority)
SELECT id, 'country', 'equals', 'CN', TRUE, 0
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO flag_rules (flag_id, attribute, operator, expected_value, return_value, priority)
SELECT id, 'plan', 'in', '["pro","enterprise"]', TRUE, 10
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO operation_histories (flag_id, actor, action, summary)
SELECT id, 'local-admin', 'CREATE_FLAG', 'seed: create checkout_v2@development'
FROM flags WHERE key = 'checkout_v2' AND environment = 'development';

INSERT INTO operation_histories (flag_id, actor, action, summary)
SELECT id, 'local-admin', 'CREATE_FLAG', 'seed: create checkout_v2@production (disabled)'
FROM flags WHERE key = 'checkout_v2' AND environment = 'production';

INSERT INTO operation_histories (flag_id, actor, action, summary)
SELECT id, 'local-admin', 'CREATE_FLAG', 'seed: create dark_mode@development'
FROM flags WHERE key = 'dark_mode' AND environment = 'development';
