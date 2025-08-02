-- Sample data for Operion PostgreSQL database
-- This file contains INSERT statements to populate the database with example workflows
--
-- IMPORTANT: Template Syntax
-- Operion uses Go template expressions with {{ }} delimiters:
-- ✅ Correct: "{{.step_results.fetch_price.body}}"
-- ✅ Correct: "{{.variables.api_base_url}}/users"
-- ✅ Correct: "Bitcoin price: ${{.step_results.transform_price.result}} USD"
-- ❌ Wrong:   "step_results.fetch_price.body"
--
-- String concatenation is done within template strings

-- ============================================================================
-- Bitcoin Price Monitoring Workflow
-- ============================================================================

-- Insert the main workflow
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

-- Insert workflow trigger (schedule-based)
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

-- Insert workflow steps
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
) VALUES 
(
    'bitcoin-step-fetch-price',
    'bitcoin-price-monitor',
    'fetch_price',
    'Fetch Bitcoin Price',
    'http_request',
    '{"url": "https://api.coinpaprika.com/v1/tickers/btc-bitcoin", "method": "GET", "headers": {"Accept": "application/json"}, "retries": {"attempts": 3, "delay": 1000}}',
    'transform_price',
    'log_error',
    true
),
(
    'bitcoin-step-transform-price',
    'bitcoin-price-monitor',
    'transform_price',
    'Extract Price Data',
    'transform',
    '{"expression": "{{.quotes.USD.price}}"}',
    'check_threshold',
    'log_error',
    true
),
(
    'bitcoin-step-check-threshold',
    'bitcoin-price-monitor',
    'check_threshold',
    'Check Price Threshold',
    'transform',
    '{"expression": "{{$price := .}}{{if gt (abs (sub $price 50000)) 1000}}{\"alert\": true, \"price\": {{$price}}, \"message\": \"Bitcoin price is {{$price}} USD\"}{{else}}{\"alert\": false, \"price\": {{$price}}}{{end}}", "input": "steps.transform_price.result"}',
    'send_alert',
    'log_price',
    true
),
(
    'bitcoin-step-send-alert',
    'bitcoin-price-monitor',
    'send_alert',
    'Send Price Alert',
    'http_request',
    '{"url": "{{.variables.webhook_url}}", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"text\": \"🚨 Bitcoin Alert: {{.steps.check_threshold.result.message}}\"}", "retries": {"attempts": 2, "delay": 500}}',
    'log_success',
    'log_error',
    true
),
(
    'bitcoin-step-log-price',
    'bitcoin-price-monitor',
    'log_price',
    'Log Current Price',
    'log',
    '{"message": "Bitcoin price: ${{.steps.transform_price.result}} USD", "level": "info"}',
    null,
    null,
    true
),
(
    'bitcoin-step-log-success',
    'bitcoin-price-monitor',
    'log_success',
    'Log Alert Sent',
    'log',
    '{"message": "Alert sent successfully for Bitcoin price: ${{.steps.check_threshold.result.price}} USD", "level": "info"}',
    null,
    null,
    true
),
(
    'bitcoin-step-log-error',
    'bitcoin-price-monitor',
    'log_error',
    'Log Error',
    'log',
    '{"message": "Error in Bitcoin price monitoring: " & error.message, "level": "error"}',
    null,
    null,
    true
);

-- ============================================================================
-- User Registration Processing Workflow
-- ============================================================================

-- Insert user registration workflow
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
    'user-registration-flow',
    'User Registration Processing',
    'Processes new user registrations with validation, welcome email, and account setup',
    '{"api_base_url": "https://api.myapp.com", "email_service_url": "https://email.myapp.com", "welcome_template": "welcome-new-user"}',
    'active',
    '{"version": "2.1", "environment": "production", "tags": ["user-management", "registration", "email"]}',
    'user-team',
    NOW(),
    NOW()
);

-- Insert webhook trigger for user registration
INSERT INTO workflow_triggers (
    id,
    workflow_id,
    name,
    description,
    trigger_id,
    configuration
) VALUES (
    'user-reg-webhook',
    'user-registration-flow',
    'Registration Webhook',
    'Triggered when a new user registers via webhook',
    'webhook',
    '{"path": "/webhooks/user-registered", "method": "POST", "workflow_id": "user-registration-flow"}'
);

-- Insert user registration workflow steps
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
) VALUES 
(
    'user-step-validate',
    'user-registration-flow',
    'validate_user',
    'Validate User Data',
    'transform',
    '{"expression": "$ ~> |$| {\"valid\": $exists($.email) and $exists($.name) and $length($.email) > 5, \"user\": $} |", "input": "trigger.webhook.body"}',
    'create_account',
    'log_validation_error',
    true
),
(
    'user-step-create-account',
    'user-registration-flow',
    'create_account',
    'Create User Account',
    'http_request',
    '{"url": "variables.api_base_url & \"/users\"", "method": "POST", "headers": {"Content-Type": "application/json", "Authorization": "Bearer " & env.API_TOKEN}, "body": "steps.validate_user.result.user", "retries": {"attempts": 3, "delay": 2000}}',
    'send_welcome_email',
    'log_account_error',
    true
),
(
    'user-step-send-welcome',
    'user-registration-flow',
    'send_welcome_email',
    'Send Welcome Email',
    'http_request',
    '{"url": "variables.email_service_url & \"/send\"", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"to\": \"" & steps.validate_user.result.user.email & "\", \"template\": \"" & variables.welcome_template & "\", \"data\": {\"name\": \"" & steps.validate_user.result.user.name & "\", \"user_id\": \"" & steps.create_account.body.id & "\"}}", "retries": {"attempts": 2, "delay": 1000}}',
    'log_registration_complete',
    'log_email_error',
    true
),
(
    'user-step-log-complete',
    'user-registration-flow',
    'log_registration_complete',
    'Log Registration Complete',
    'log',
    '{"message": "User registration completed successfully for " & steps.validate_user.result.user.email & " (ID: " & steps.create_account.body.id & ")", "level": "info"}',
    null,
    null,
    true
),
(
    'user-step-log-validation-error',
    'user-registration-flow',
    'log_validation_error',
    'Log Validation Error',
    'log',
    '{"message": "User registration validation failed: " & trigger.webhook.body, "level": "error"}',
    null,
    null,
    true
),
(
    'user-step-log-account-error',
    'user-registration-flow',
    'log_account_error',
    'Log Account Creation Error',
    'log',
    '{"message": "Failed to create user account for " & steps.validate_user.result.user.email & ": " & error.message, "level": "error"}',
    'send_admin_alert',
    null,
    true
),
(
    'user-step-log-email-error',
    'user-registration-flow',
    'log_email_error',
    'Log Email Error',
    'log',
    '{"message": "Failed to send welcome email to " & steps.validate_user.result.user.email & ": " & error.message, "level": "warn"}',
    null,
    null,
    true
),
(
    'user-step-admin-alert',
    'user-registration-flow',
    'send_admin_alert',
    'Send Admin Alert',
    'http_request',
    '{"url": "variables.webhook_url", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"text\": \"⚠️ User registration system error - manual intervention may be required\"}", "retries": {"attempts": 1, "delay": 500}}',
    null,
    null,
    true
);

-- ============================================================================
-- Daily System Health Check Workflow
-- ============================================================================

-- Insert system health check workflow
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
    'daily-health-check',
    'Daily System Health Check',
    'Performs daily health checks on critical system components and generates status report',
    '{"services": ["database", "redis", "elasticsearch"], "slack_webhook": "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK", "health_endpoints": ["https://api.myapp.com/health", "https://auth.myapp.com/health"]}',
    'active',
    '{"version": "1.5", "environment": "production", "tags": ["monitoring", "health-check", "operations"]}',
    'devops-team',
    NOW(),
    NOW()
);

-- Insert daily schedule trigger
INSERT INTO workflow_triggers (
    id,
    workflow_id,
    name,
    description,
    trigger_id,
    configuration
) VALUES (
    'health-check-daily',
    'daily-health-check',
    'Daily at 8 AM',
    'Runs health check every day at 8:00 AM UTC',
    'schedule',
    '{"cron": "0 8 * * *", "workflow_id": "daily-health-check", "enabled": true}'
);

-- Insert health check workflow steps
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
) VALUES 
(
    'health-step-check-api',
    'daily-health-check',
    'check_api_health',
    'Check API Health',
    'http_request',
    '{"url": "https://api.myapp.com/health", "method": "GET", "headers": {"Accept": "application/json"}, "retries": {"attempts": 2, "delay": 5000}}',
    'check_database',
    'log_api_error',
    true
),
(
    'health-step-check-database',
    'daily-health-check',
    'check_database',
    'Check Database Health',
    'http_request',
    '{"url": "https://api.myapp.com/health/database", "method": "GET", "headers": {"Accept": "application/json"}, "retries": {"attempts": 2, "delay": 3000}}',
    'generate_report',
    'log_db_error',
    true
),
(
    'health-step-generate-report',
    'daily-health-check',
    'generate_report',
    'Generate Health Report',
    'transform',
    '{"expression": "{\"timestamp\": $now(), \"api_status\": steps.check_api_health.status = 200 ? \"healthy\" : \"unhealthy\", \"database_status\": steps.check_database.status = 200 ? \"healthy\" : \"unhealthy\", \"overall_health\": (steps.check_api_health.status = 200 and steps.check_database.status = 200) ? \"all_systems_healthy\" : \"issues_detected\"}"}',
    'send_report',
    'send_error_report',
    true
),
(
    'health-step-send-report',
    'daily-health-check',
    'send_report',
    'Send Health Report',
    'http_request',
    '{"url": "variables.slack_webhook", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"text\": \"📊 Daily Health Check Report\\n• API Status: " & steps.generate_report.result.api_status & "\\n• Database Status: " & steps.generate_report.result.database_status & "\\n• Overall: " & steps.generate_report.result.overall_health & "\\n• Timestamp: " & steps.generate_report.result.timestamp & "\"}", "retries": {"attempts": 2, "delay": 1000}}',
    'log_report_sent',
    'log_report_error',
    true
),
(
    'health-step-log-api-error',
    'daily-health-check',
    'log_api_error',
    'Log API Error',
    'log',
    '{"message": "API health check failed: " & error.message, "level": "error"}',
    'check_database',
    null,
    true
),
(
    'health-step-log-db-error',
    'daily-health-check',
    'log_db_error',
    'Log Database Error',
    'log',
    '{"message": "Database health check failed: " & error.message, "level": "error"}',
    'send_error_report',
    null,
    true
),
(
    'health-step-send-error-report',
    'daily-health-check',
    'send_error_report',
    'Send Error Report',
    'http_request',
    '{"url": "variables.slack_webhook", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"text\": \"🚨 Daily Health Check - ISSUES DETECTED\\nSome system components are not responding normally. Please check logs for details.\"}", "retries": {"attempts": 3, "delay": 2000}}',
    'log_error_report_sent',
    'log_report_error',
    true
),
(
    'health-step-log-report-sent',
    'daily-health-check',
    'log_report_sent',
    'Log Report Sent',
    'log',
    '{"message": "Daily health check report sent successfully", "level": "info"}',
    null,
    null,
    true
),
(
    'health-step-log-report-error',
    'daily-health-check',
    'log_report_error',
    'Log Report Error',
    'log',
    '{"message": "Failed to send health check report: " & error.message, "level": "error"}',
    null,
    null,
    true
);

-- ============================================================================
-- Summary of inserted data
-- ============================================================================

-- Verify data insertion with summary queries
SELECT 
    'Workflows' as entity_type,
    COUNT(*) as total_count,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count
FROM workflows
WHERE deleted_at IS NULL

UNION ALL

SELECT 
    'Triggers' as entity_type,
    COUNT(*) as total_count,
    NULL as active_count
FROM workflow_triggers

UNION ALL

SELECT 
    'Steps' as entity_type,
    COUNT(*) as total_count,
    COUNT(CASE WHEN enabled = true THEN 1 END) as enabled_count
FROM workflow_steps;

-- Show workflow summary
SELECT 
    w.name as workflow_name,
    w.status,
    COUNT(DISTINCT wt.id) as trigger_count,
    COUNT(DISTINCT ws.id) as step_count,
    w.owner,
    w.created_at
FROM workflows w
LEFT JOIN workflow_triggers wt ON w.id = wt.workflow_id
LEFT JOIN workflow_steps ws ON w.id = ws.workflow_id
WHERE w.deleted_at IS NULL
GROUP BY w.id, w.name, w.status, w.owner, w.created_at
ORDER BY w.created_at DESC;