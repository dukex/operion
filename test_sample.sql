-- Test the first workflow insertion from sample_data.sql

INSERT INTO workflows (
    id, 
    name, 
    description, 
    variables, 
    status, 
    metadata, 
    owner, 
    created_at, 
    updated_at
) VALUES (
    'bitcoin-price-monitor',
    'Bitcoin Price Monitor',
    'Monitors Bitcoin price every 5 minutes and sends alerts when price changes significantly',
    '{"api_base_url": "https://api.coinpaprika.com", "alert_threshold": 1000, "webhook_url": "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"}',
    'active',
    '{"version": "1.0", "environment": "production", "tags": ["cryptocurrency", "monitoring", "alerts"]}',
    'system',
    NOW(),
    NOW()
);

-- Test the first trigger insertion
INSERT INTO workflow_triggers (
    id,
    workflow_id,
    name,
    description,
    trigger_id,
    configuration
) VALUES (
    'bitcoin-trigger-schedule',
    'bitcoin-price-monitor',
    'Every 5 minutes',
    'Triggers Bitcoin price check every 5 minutes',
    'schedule',
    '{"cron": "*/5 * * * *", "workflow_id": "bitcoin-price-monitor", "enabled": true}'
);

-- Test the first step insertion with fixed JSON
INSERT INTO workflow_steps (
    id,
    workflow_id,
    uid,
    name,
    action_id,
    configuration,
    on_success,
    on_failure,
    enabled
) VALUES (
    'bitcoin-step-fetch-price',
    'bitcoin-price-monitor',
    'fetch_price',
    'Fetch Bitcoin Price',
    'http_request',
    '{"url": "https://api.coinpaprika.com/v1/tickers/btc-bitcoin", "method": "GET", "headers": {"Accept": "application/json"}, "retries": {"attempts": 3, "delay": 1000}}',
    'transform_price',
    'log_error',
    true
);