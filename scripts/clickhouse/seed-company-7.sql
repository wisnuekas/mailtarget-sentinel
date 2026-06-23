-- Seed: company_id 7 / sub_account_id 239
-- Re-run: make clickhouse-seed-7
--
-- injection_time spread across the last ~4 minutes so data stays
-- inside the 5-minute detection window for longer (survives cron ticks).
--
-- Expected: ~8% bounce rate -> ANOMALY (threshold 5%, min volume 100)
-- sending_ip 10.10.0.201 grouped for at-risk IP detection

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp, sending_ip)
SELECT
    concat('seed-7-239-inj-', toString(number)),
    7, 'injection',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    239, 'default', '', 0,
    now() - INTERVAL (30 + (number % 210)) SECOND,
    '10.10.0.201'
FROM numbers(100);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp, sending_ip)
SELECT
    concat('seed-7-239-bnc-', toString(number)),
    7, 'bounce',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    239, 'default', '550 Mailbox unavailable', 10,
    now() - INTERVAL (30 + (number % 210)) SECOND,
    '10.10.0.201'
FROM numbers(7);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp, sending_ip)
SELECT
    concat('seed-7-239-spam-', toString(number)),
    7, 'bounce',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'yahoo.com',
    239, 'default', 'Spam Block', 51,
    now() - INTERVAL (30 + (number % 210)) SECOND,
    '10.10.0.201'
FROM numbers(1);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp, sending_ip)
SELECT
    concat('seed-7-239-dlv-', toString(number)),
    7, 'delivery',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    239, 'default', '', 0,
    now() - INTERVAL (30 + (number % 210)) SECOND,
    '10.10.0.201'
FROM numbers(92);
