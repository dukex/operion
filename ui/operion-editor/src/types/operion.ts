export type WorkflowStatus = "active" | "inactive" | "paused" | "error";

// export interface ConfigSchemaProperty {
//   type: "string" | "number" | "boolean" | "object" | "array";
//   required?: boolean;
//   description?: string;
//   enum?: string[];
//   default?: any;
//   items?: ConfigSchemaProperty; // For array type
//   properties?: Record<string, ConfigSchemaProperty>; // For object type
// }

// export interface ActionRegistryItem {
//   type: string;
//   name: string;
//   description: string;
//   config_schema: Record<string, ConfigSchemaProperty>;
// }

// export interface TriggerRegistryItem {
//   type: string;
//   name: string;
//   description: string;
//   config_schema: Record<string, ConfigSchemaProperty>;
// }

export interface TriggerItem {
  id: string;
  type: string;
  configuration: {
    cron?: string; // For schedule triggers
    [key: string]: unknown; // Additional trigger-specific configuration
  };
}

export interface ConditionalExpression {
  language: "javascript" | "cel" | "simple" | "";
  expression: string;
}

export interface WorkflowStep {
  id: string;
  action_id: string;
  uid: string;
  name: string;
  description: string;
  configuration: Record<string, unknown>;
  conditional?: ConditionalExpression;
  on_success?: string | null;
  on_failure?: string | null;
  enabled: boolean;
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  workflow_triggers: TriggerItem[];
  steps: WorkflowStep[];
  variables: Record<string, unknown>;
  status: WorkflowStatus;
  metadata: Record<string, unknown>;
  owner: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

// export interface CreateWorkflowPayload {
//   name: string;
//   description: string;
//   triggers?: Partial<TriggerItem>[]; // IDs are not provided by client initially for triggers array
//   steps?: Omit<WorkflowStep, "id" | "action"> &
//     { action: Omit<ActionItem, "id"> }[]; // IDs are auto-generated for steps and actions
//   variables?: Record<string, any>;
//   status?: WorkflowStatus;
//   metadata?: Record<string, any>;
//   owner?: string;
// }

// export type UpdateWorkflowPayload = Partial<
//   Omit<CreateWorkflowPayload, "triggers" | "steps">
// > & {
//   triggers?: Partial<TriggerItem>[];
//   steps?: (Omit<WorkflowStep, "id" | "action"> & {
//     action: Omit<ActionItem, "id">;
//   })[];
// };

// export interface UpdateWorkflowStepsPayloadEntry {
//   id: string; // Added step ID
//   name: string;
//   action: Omit<ActionItem, "id">; // ID auto-generated for action within step
//   conditional?: ConditionalExpression;
//   on_success?: string | null;
//   on_failure?: string | null;
//   enabled: boolean;
// }

// export type UpdateWorkflowStepsPayload = UpdateWorkflowStepsPayloadEntry[];

export interface ApiErrorProblem {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance: string;
}
