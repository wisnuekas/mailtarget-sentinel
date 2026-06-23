-- Seed: company_id 287 / sub_account_id 4302
-- Re-run: make clickhouse-seed-287
--
-- injection_time spread across the last ~4 minutes so data stays
-- inside the 5-minute detection window for longer (survives cron ticks).
--
-- Expected: ~8% bounce rate → ANOMALY (threshold 5%, min volume 100)

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-287-4302-inj-', toString(number)),
    287, 'injection',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    4302, 'default', '', 0,
    now() - INTERVAL (30 + (number % 210)) SECOND
FROM numbers(250);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-287-4302-bnc-', toString(number)),
    287, 'bounce',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    4302, 'default', '550 Mailbox unavailable', 10,
    now() - INTERVAL (30 + (number % 210)) SECOND
FROM numbers(17);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-287-4302-spam-', toString(number)),
    287, 'bounce',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'yahoo.com',
    4302, 'default', 'Spam Block', 51,
    now() - INTERVAL (30 + (number % 210)) SECOND
FROM numbers(3);

INSERT INTO default.event
    (event_id, company_id, type, injection_time, sending_domain, recipient_domain,
     sub_account_id, ip_pool, reason, bounce_classification_code, timestamp)
SELECT
    concat('seed-287-4302-dlv-', toString(number)),
    287, 'delivery',
    now() - INTERVAL (30 + (number % 210)) SECOND,
    'mail.mailtarget.co', 'gmail.com',
    4302, 'default', '', 0,
    now() - INTERVAL (30 + (number % 210)) SECOND
FROM numbers(220);
