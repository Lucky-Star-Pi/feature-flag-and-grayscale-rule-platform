DELETE FROM operation_histories;
DELETE FROM flag_rules;
DELETE FROM flags WHERE key IN ('checkout_v2', 'dark_mode');
