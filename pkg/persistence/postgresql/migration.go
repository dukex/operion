package postgresql

func migrations() map[int]string {
	return map[int]string{
		1: `
			-- Create workflows table
			CREATE TABLE workflows (
				id UUID PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				description TEXT NOT NULL,
				variables JSONB,
				status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'inactive', 'paused', 'error')),
				metadata JSONB,
				owner VARCHAR(255),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				deleted_at TIMESTAMP WITH TIME ZONE
			);

			CREATE INDEX idx_workflows_status ON workflows(status);
			CREATE INDEX idx_workflows_owner ON workflows(owner);
			CREATE INDEX idx_workflows_created_at ON workflows(created_at);
			CREATE INDEX idx_workflows_deleted_at ON workflows(deleted_at);

			-- Create workflow_triggers table
			CREATE TABLE workflow_triggers (
				id UUID PRIMARY KEY,
				workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				description TEXT NOT NULL,
				source_id VARCHAR(255),
				event_type VARCHAR(255) NOT NULL,
				provider_id VARCHAR(255) NOT NULL,
				configuration JSONB,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);

			CREATE INDEX idx_workflow_triggers_workflow_id ON workflow_triggers(workflow_id);
			CREATE INDEX idx_workflow_triggers_event_type ON workflow_triggers(event_type);
			CREATE INDEX idx_workflow_triggers_provider_id ON workflow_triggers(provider_id);
			CREATE INDEX idx_workflow_triggers_source_id ON workflow_triggers(source_id);

			-- Create workflow_steps table
			CREATE TABLE workflow_steps (
				id UUID PRIMARY KEY,
				workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
				uid VARCHAR(255) NOT NULL,
				name VARCHAR(255) NOT NULL,
				action_id VARCHAR(255) NOT NULL,
				configuration JSONB,
				conditional_language VARCHAR(50),
				conditional_expression TEXT,
				on_success VARCHAR(255),
				on_failure VARCHAR(255),
				enabled BOOLEAN NOT NULL DEFAULT true,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);

			CREATE INDEX idx_workflow_steps_workflow_id ON workflow_steps(workflow_id);
			CREATE INDEX idx_workflow_steps_action_id ON workflow_steps(action_id);
			CREATE INDEX idx_workflow_steps_uid ON workflow_steps(uid);

			-- Unique constraint on workflow_id + uid for steps
			CREATE UNIQUE INDEX idx_workflow_steps_workflow_uid ON workflow_steps(workflow_id, uid);
		`,
	}
}
