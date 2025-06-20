import { useState, useEffect, useCallback } from "react";

import { getWorkflows } from "@/lib/api-client";
import { Loader2, AlertTriangle, PlusCircle } from "lucide-react";
import type {
  Workflow,
  // CreateWorkflowPayload,
  ApiErrorProblem,
} from "@/types/operion";

import { Button } from "@/components/ui/Button";
import { WorkflowCard } from "@/components/workflow/WorkflowCard";

export default function Home() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // const [workflowToDelete, setWorkflowToDelete] = useState<string | null>(null);

  const fetchWorkflows = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await getWorkflows();
      setWorkflows(data);
    } catch (err) {
      const apiError = err as ApiErrorProblem;
      console.error("Failed to fetch workflows:", apiError);
      setError(
        apiError.detail || "Failed to load workflows. Please try again."
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  return (
    <>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold font-headline">My Workflows</h1>
      </div>

      {isLoading && (
        <div className="flex justify-center items-center h-64">
          <Loader2 className="h-12 w-12 animate-spin text-primary" />
          <p className="ml-4 text-lg text-muted-foreground">
            Loading workflows...
          </p>
        </div>
      )}

      {error && !isLoading && (
        <div className="flex flex-col items-center justify-center h-64 bg-destructive/10 p-6 rounded-lg border border-destructive">
          <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
          <p className="text-xl font-semibold text-destructive mb-2">
            Oops! Something went wrong.
          </p>
          <p className="text-destructive/80 text-center mb-6">{error}</p>
          <Button onClick={fetchWorkflows} variant="destructive">
            Try Again
          </Button>
        </div>
      )}

      {!isLoading && !error && workflows.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {workflows.map((workflow) => (
            <WorkflowCard
              key={workflow.id}
              workflow={workflow}
              // onDelete={() => setWorkflowToDelete(workflow.id)}
            />
          ))}
        </div>
      )}
    </>
  );
}
