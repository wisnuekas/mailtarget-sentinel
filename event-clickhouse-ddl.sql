-- `default`.event definition

CREATE TABLE default.event
(

    `event_id` String,

    `company_id` Int32,

    `type` String,

    `injection_time` DateTime,

    `sending_domain` String,

    `recipient_domain` String,

    `sub_account_id` Int32,

    `ip_pool` String,

    `reason` String,

    `bounce_classification_code` Int32,

    `timestamp` DateTime,

    `amp_enabled` Nullable(Bool),

    `click_tracking` Nullable(Bool),

    `delivery_method` Nullable(String),

    `error_code` Nullable(Int32),

    `friendly_from` Nullable(String),

    `geo_ip` Map(String,
 String),

    `ip_address` Nullable(String),

    `mailbox_provider` Nullable(String),

    `mailbox_provider_region` Nullable(String),

    `envelope_id` Nullable(String),

    `message_from` Nullable(String),

    `message_size` Nullable(Int32),

    `num_retries` Nullable(Int32),

    `open_tracking` Nullable(Bool),

    `queue_time` Nullable(Int32),

    `recipient_meta` Map(String,
 String),

    `recipient_to` Nullable(String),

    `raw_reason` Nullable(String),

    `raw_recipient_to` Nullable(String),

    `recipient_type` Nullable(String),

    `received_method` Nullable(String),

    `routing_domain` Nullable(String),

    `scheduled_time` Nullable(Int32),

    `sending_ip` Nullable(String),

    `subject` Nullable(String),

    `target_link_name` Nullable(String),

    `target_link_url` Nullable(String),

    `template_id` Nullable(String),

    `template_version` Nullable(Int32),

    `transactional` Nullable(Int32),

    `transmission_id` Nullable(String),

    `user_agent` Nullable(String),

    `user_agent_parsed` Map(String,
 String)
)
ENGINE = ReplacingMergeTree
PRIMARY KEY (injection_time,
 type,
 bounce_classification_code,
 company_id,
 ip_pool,
 sub_account_id,
 reason,
 recipient_domain,
 event_id)
ORDER BY (injection_time,
 type,
 bounce_classification_code,
 company_id,
 ip_pool,
 sub_account_id,
 reason,
 recipient_domain,
 event_id)
SETTINGS index_granularity = 8192;