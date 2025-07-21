import type {
  Workflow,
  //   ActionRegistryItem,
  //   TriggerRegistryItem,
  //   CreateWorkflowPayload,
  //   UpdateWorkflowPayload,
  //   UpdateWorkflowStepsPayload,
  ApiErrorProblem,
} from "@/types/operion";

const API_BASE_URL = process.env.API_BASE_URL;

async function handleResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) {
    return null as T;
  }

  const contentType = response.headers.get("content-type");
  const data = await response.json();

  if (!response.ok) {
    // Attempt to parse as ApiErrorProblem if JSON, otherwise use text
    const errorDetail =
      contentType && contentType.includes("application/json")
        ? data
        : { detail: data || response.statusText };
    const error: ApiErrorProblem = {
      type: errorDetail.type || "api_error",
      title: errorDetail.title || "API Error",
      status: response.status,
      detail:
        errorDetail.detail || `Request failed with status ${response.status}`,
      instance: errorDetail.instance || "",
    };
    console.error("API Error:", error);
    throw error;
  }
  return data as T;
}

//   export async function getApiHealth(): Promise<string> {
//     const response = await fetch(`${API_BASE_URL}/`);
//     return handleResponse<string>(response);
//   }

export async function getWorkflows(): Promise<Workflow[]> {
  const response = await fetch(`${API_BASE_URL}/workflows`);
  return handleResponse<Workflow[]>(response);
}

export async function getWorkflowById(id: string): Promise<Workflow> {
  const response = await fetch(`${API_BASE_URL}/workflows/${id}`);
  return handleResponse<Workflow>(response);
}

//   export async function createWorkflow(payload: CreateWorkflowPayload): Promise<Workflow> {
//     const response = await fetch(`${API_BASE_URL}/workflows`, {
//       method: 'POST',
//       headers: { 'Content-Type': 'application/json' },
//       body: JSON.stringify(payload),
//     });
//     return handleResponse<Workflow>(response);
//   }

//   export async function updateWorkflow(id: string, payload: UpdateWorkflowPayload): Promise<Workflow> {
//     const response = await fetch(`${API_BASE_URL}/workflows/${id}`, {
//       method: 'PATCH',
//       headers: { 'Content-Type': 'application/json' },
//       body: JSON.stringify(payload),
//     });
//     return handleResponse<Workflow>(response);
//   }

//   export async function deleteWorkflow(id: string): Promise<void> {
//     const response = await fetch(`${API_BASE_URL}/workflows/${id}`, {
//       method: 'DELETE',
//     });
//     await handleResponse<void>(response);
//   }

//   export async function updateWorkflowSteps(id: string, payload: UpdateWorkflowStepsPayload): Promise<Workflow['steps']> {
//     const response = await fetch(`${API_BASE_URL}/workflows/${id}/steps`, {
//       method: 'PATCH',
//       headers: { 'Content-Type': 'application/json' },
//       body: JSON.stringify(payload),
//     });
//     return handleResponse<Workflow['steps']>(response);
//   }

  export async function getAvailableActions(): Promise<ActionRegistryItem[]> {
    const response = await fetch(`${API_BASE_URL}/registry/actions`);
    return handleResponse<ActionRegistryItem[]>(response);
  }

  export async function getAvailableTriggers(): Promise<TriggerRegistryItem[]> {
    const response = await fetch(`${API_BASE_URL}/registry/triggers`);
    return handleResponse<TriggerRegistryItem[]>(response);
  }
