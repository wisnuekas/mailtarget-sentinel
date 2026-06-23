-- Dev seed: sample events in the last ~3 minutes.
-- Re-run anytime: make clickhouse-seed
--
-- Expected results (5m window, default thresholds):
--   company 42 / sub_account 101 → ~7.5% bounce (ANOMALY)
--   company 99 / sub_account 201 → ~10% bounce + spam (ANOMALY)
--   company 42 / sub_account 102 → ~1% bounce (healthy)

-- Healthy sub-account: 500 sent, 5 bounce, 480 delivery
INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-102-inj-', toString(number)) AS event_id,
    42 AS company_id,
    'injection' AS type,
    now() - INTERVAL 2 MINUTE AS injection_time,
    'mail.example.com' AS sending_domain,
    'gmail.com' AS recipient_domain,
    102 AS sub_account_id,
    'default' AS ip_pool,
    '' AS reason,
    0 AS bounce_classification_code,
    now() - INTERVAL 2 MINUTE AS timestamp
FROM numbers(500);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-102-bnc-', toString(number)),
    42, 'bounce',
    now() - INTERVAL 2 MINUTE,
    'mail.example.com', 'gmail.com',
    102, 'default', '550 User unknown', 10,
    now() - INTERVAL 2 MINUTE
FROM numbers(5);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-102-dlv-', toString(number)),
    42, 'delivery',
    now() - INTERVAL 2 MINUTE,
    'mail.example.com', 'gmail.com',
    102, 'default', '', 0,
    now() - INTERVAL 2 MINUTE
FROM numbers(480);

-- Anomaly sub-account: 200 sent, 15 bounce (~7.5%)
INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-101-inj-', toString(number)),
    42, 'injection',
    now() - INTERVAL 3 MINUTE,
    'mail.example.com', 'yahoo.com',
    101, 'default', '', 0,
    now() - INTERVAL 3 MINUTE
FROM numbers(200);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-101-bnc-', toString(number)),
    42, 'bounce',
    now() - INTERVAL 3 MINUTE,
    'mail.example.com', 'yahoo.com',
    101, 'default', '550 Mailbox unavailable', 10,
    now() - INTERVAL 3 MINUTE
FROM numbers(15);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-101-dlv-', toString(number)),
    42, 'delivery',
    now() - INTERVAL 3 MINUTE,
    'mail.example.com', 'yahoo.com',
    101, 'default', '', 0,
    now() - INTERVAL 3 MINUTE
FROM numbers(170);

-- Second company anomaly: 150 sent, 12 bounce + 3 spam (classification 51)
INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-201-inj-', toString(number)),
    99, 'injection',
    now() - INTERVAL 1 MINUTE,
    'promo.example.com', 'outlook.com',
    201, 'pool-2', '', 0,
    now() - INTERVAL 1 MINUTE
FROM numbers(150);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-201-bnc-', toString(number)),
    99, 'bounce',
    now() - INTERVAL 1 MINUTE,
    'promo.example.com', 'outlook.com',
    201, 'pool-2', '550 Hard bounce', 10,
    now() - INTERVAL 1 MINUTE
FROM numbers(9);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-201-spam-', toString(number)),
    99, 'bounce',
    now() - INTERVAL 1 MINUTE,
    'promo.example.com', 'outlook.com',
    201, 'pool-2', 'Spam Block', 51,
    now() - INTERVAL 1 MINUTE
FROM numbers(3);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-201-dlv-', toString(number)),
    99, 'delivery',
    now() - INTERVAL 1 MINUTE,
    'promo.example.com', 'outlook.com',
    201, 'pool-2', '', 0,
    now() - INTERVAL 1 MINUTE
FROM numbers(130);
