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

		`,
		2: `
			-- Migration 2: Node-based workflow architecture with published versioning
			
			-- Add new columns to workflows table for published versioning
			ALTER TABLE workflows 
				ADD COLUMN published_id UUID,
				ADD COLUMN parent_id UUID,
				ADD COLUMN published_at TIMESTAMP WITH TIME ZONE;

			-- Remove the CHECK constraint on status to support draft/published
			ALTER TABLE workflows DROP CONSTRAINT workflows_status_check;

			-- Add indexes for new columns
			CREATE INDEX idx_workflows_published_id ON workflows(published_id);
			CREATE INDEX idx_workflows_parent_id ON workflows(parent_id);
			CREATE INDEX idx_workflows_published_at ON workflows(published_at);

			-- Create workflow_nodes table
			CREATE TABLE workflow_nodes (
				workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
				id VARCHAR(255) NOT NULL,
				node_type VARCHAR(255) NOT NULL,
				category VARCHAR(50) NOT NULL DEFAULT 'action',  -- 'action' or 'trigger'
				config JSONB DEFAULT '{}',
				position_x INT DEFAULT 0,
				position_y INT DEFAULT 0,
				name VARCHAR(255) NOT NULL,
				enabled BOOLEAN NOT NULL DEFAULT true,
				source_id VARCHAR(255),      -- For trigger nodes only
				provider_id VARCHAR(255),    -- For trigger nodes only  
				event_type VARCHAR(255),     -- For trigger nodes only
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				PRIMARY KEY (workflow_id, id)
			);

			CREATE INDEX idx_workflow_nodes_workflow_id ON workflow_nodes(workflow_id);
			CREATE INDEX idx_workflow_nodes_type ON workflow_nodes(node_type);
			CREATE INDEX idx_workflow_nodes_category ON workflow_nodes(category);
			CREATE INDEX idx_workflow_nodes_source_id ON workflow_nodes(source_id);
			CREATE INDEX idx_workflow_nodes_provider_id ON workflow_nodes(provider_id);

			-- Create workflow_connections table
			CREATE TABLE workflow_connections (
				workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
				id VARCHAR(255) NOT NULL,
				source_node_id VARCHAR(255) NOT NULL,
				source_port VARCHAR(255) NOT NULL,
				target_node_id VARCHAR(255) NOT NULL,
				target_port VARCHAR(255) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				PRIMARY KEY (workflow_id, id)
			);

			CREATE INDEX idx_workflow_connections_workflow_id ON workflow_connections(workflow_id);
			CREATE INDEX idx_workflow_connections_source ON workflow_connections(source_node_id);
			CREATE INDEX idx_workflow_connections_target ON workflow_connections(target_node_id);
			CREATE UNIQUE INDEX idx_workflow_connections_unique ON workflow_connections(workflow_id, source_node_id, source_port, target_node_id, target_port);

			-- Create execution_contexts table
			CREATE TABLE execution_contexts (
				id VARCHAR(255) PRIMARY KEY,
				published_workflow_id UUID NOT NULL REFERENCES workflows(id),
				status VARCHAR(50) NOT NULL,
				trigger_data JSONB DEFAULT '{}',
				variables JSONB DEFAULT '{}',
				error_message TEXT,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				completed_at TIMESTAMP WITH TIME ZONE
			);

			CREATE INDEX idx_execution_contexts_workflow_id ON execution_contexts(published_workflow_id);
			CREATE INDEX idx_execution_contexts_status ON execution_contexts(status);
			CREATE INDEX idx_execution_contexts_created_at ON execution_contexts(created_at);

			-- Create node_executions table (tracks individual node results)
			CREATE TABLE node_executions (
				id VARCHAR(255) PRIMARY KEY,
				execution_id VARCHAR(255) NOT NULL REFERENCES execution_contexts(id) ON DELETE CASCADE,
				node_id VARCHAR(255) NOT NULL,
				status VARCHAR(50) NOT NULL,
				input_data JSONB DEFAULT '{}',
				output_data JSONB DEFAULT '{}',
				error_message TEXT,
				started_at TIMESTAMP WITH TIME ZONE,
				completed_at TIMESTAMP WITH TIME ZONE,
				duration_ms BIGINT
			);

			CREATE INDEX idx_node_executions_execution_id ON node_executions(execution_id);
			CREATE INDEX idx_node_executions_node_id ON node_executions(node_id);
			CREATE INDEX idx_node_executions_status ON node_executions(status);
			CREATE INDEX idx_node_executions_started_at ON node_executions(started_at);
			CREATE UNIQUE INDEX idx_node_executions_unique ON node_executions(execution_id, node_id, started_at);
		`,
		3: `
			-- Migration 3: Port-based node architecture
			
			-- Create workflow_ports table: Clean normalization, ports belong to nodes
			CREATE TABLE workflow_ports (
				id VARCHAR(255) PRIMARY KEY,           -- "{node_id}:{port_name}"
				node_id VARCHAR(255) NOT NULL,        -- References workflow_nodes.id
				name VARCHAR(255) NOT NULL,            -- Port name within the node
				direction VARCHAR(10) NOT NULL,        -- "input" or "output"
				description TEXT,
				data_type VARCHAR(50),
				schema JSONB,
				required BOOLEAN DEFAULT FALSE,        -- Only relevant for input ports
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				
				-- Simple constraints
				UNIQUE(node_id, name, direction),      -- One port name per direction per node
				CHECK (direction IN ('input', 'output'))
			);

			-- Create workflow_connections table: Just port-to-port relationships  
			CREATE TABLE workflow_connections_new (
				id VARCHAR(255) PRIMARY KEY,
				source_port_id VARCHAR(255) NOT NULL,  -- References workflow_ports.id
				target_port_id VARCHAR(255) NOT NULL,  -- References workflow_ports.id
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				
				-- Clean constraints
				UNIQUE(source_port_id, target_port_id) -- No duplicate connections
			);

			-- Indexes for performance
			CREATE INDEX idx_workflow_ports_node ON workflow_ports(node_id);
			CREATE INDEX idx_workflow_ports_direction ON workflow_ports(direction);
			CREATE INDEX idx_workflow_connections_new_source ON workflow_connections_new(source_port_id);
			CREATE INDEX idx_workflow_connections_new_target ON workflow_connections_new(target_port_id);
			
			-- Note: Foreign key constraints will be added after data migration
		`,
	}
}
