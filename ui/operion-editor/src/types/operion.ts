export type WorkflowStatus = "active" | "inactive" | "paused" | "error";

export interface JsonSchemaMinimal {
  type: "string" | "number" | "object" | "integer";
  required?: string[];
  default?: unknown; // Default value for the field0
  format?: string; // e.g., "date-time", "email", etc.
  title?: string;
  description?: string;
  enum?: string[];
  examples?: string[];
  example?: string;
  maximum?: number; // For numeric types
  minimum?: number; // For numeric types
  items?: JsonSchemaMinimal; // For array type
  properties?: Record<string, JsonSchemaMinimal>; // For object type
}

export interface ActionRegistryItem {
  id: string;
  type: string;
  name: string;
  description: string;
  schema: JsonSchemaMinimal;
}

export interface TriggerRegistryItem {
  id: string;
  type: string;
  name: string;
  description: string;
  schema: JsonSchemaMinimal;
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
  configuration: Record<string, unknown>;
  conditional?: ConditionalExpression;
  on_success?: string | null;
  on_failure?: string | null;
  enabled: boolean;
}

export interface TriggerItem {
  id: string;
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
