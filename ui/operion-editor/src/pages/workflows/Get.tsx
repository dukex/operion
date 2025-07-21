import { useParams } from "react-router";
import type {
  Workflow,
  ApiErrorProblem,
  ActionRegistryItem,
  TriggerRegistryItem,
} from "@/types/operion";
import { useCallback, useEffect, useState } from "react";
import {
  getAvailableActions,
  getAvailableTriggers,
  getWorkflowById,
} from "@/lib/api-client";
import { AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import {
  ReactFlow,
  MiniMap,
  Background,
  useNodesState,
  useEdgesState,
  type Node,
  addEdge,
  type Edge,
  MarkerType,
  Position,
  BackgroundVariant,
  type Connection,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { TriggerItem, WorkflowStep } from "@/types/operion";
import TriggerNode from "@/components/flow/TriggerNode";
import StepNode from "@/components/flow/StepNode";
import dagre from "@dagrejs/dagre";
import { Palette } from "@/components/flow/Pallete";
import StepOrTriggerDialog from "@/components/flow/StepOrTriggerDialog";

const nodeTypes = {
  trigger: TriggerNode,
  step: StepNode,
};

const nodeWidth = 359;
const nodeHeight = 102;

const getLayoutedElements = (
  nodes: Partial<Node>[],
  edges: Edge[],
  direction = "TB"
): { nodes: Node[]; edges: Edge[] } => {
  const isHorizontal = direction === "LR";
  dagreGraph.setGraph({ rankdir: direction });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id!, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const newNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id!);
    const newNode = {
      ...node,
      targetPosition: isHorizontal ? Position.Left : Position.Top,
      sourcePosition: isHorizontal ? Position.Right : Position.Bottom,
      position: {
        x: nodeWithPosition.x,
        y: nodeWithPosition.y,
      },
    };

    return newNode as Node;
  });

  return { nodes: newNodes, edges };
};

const withDefaultParams = (params: Edge, success: boolean): Edge => {
  return {
    animated: true,
    sourceHandle: success ? "success" : "failure",
    label: success ? "on success" : "on failure",
    style: success
      ? { stroke: "var(--color-green-500)", strokeWidth: 1 }
      : { stroke: "hsl(var(--destructive))", strokeWidth: 1 },
    markerEnd: success
      ? {
          type: MarkerType.ArrowClosed,
          width: 20,
          height: 20,
          color: "var(--color-green-500)",
        }
      : {
          type: MarkerType.ArrowClosed,
          width: 20,
          height: 20,
          color: "hsl(var(--destructive))",
        },
    ...params,
  };
};

const dagreGraph = new dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));

export default function WorkflowsGet() {
  const params = useParams();
  const id = params.id as string;

  const [workflow, setWorkflow] = useState<Workflow | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [editingStep, setEditingStep] = useState<
    (WorkflowStep & { isNew?: boolean }) | null
  >(null);
  const [triggersRegistry, setTriggerRegistry] = useState<
    TriggerRegistryItem[]
  >([]);
  const [actionsRegistry, setActionsRegistry] = useState<ActionRegistryItem[]>(
    []
  );

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) =>
        addEdge(
          withDefaultParams(
            {
              ...params,
              id: `${params.source}-${params.target}-${params.sourceHandle}`,
            },
            params["sourceHandle"] === "success"
          ),
          eds
        )
      );
    },

    [setEdges]
  );

  const handleAddItem = (id: string) => {
    // TODO: handle with item type, if it is a trigger or step
    if (!workflow) return;
    const action = actionsRegistry.find((a) => a.id === id);
    if (!action) return;

    const newStep: WorkflowStep & { isNew?: boolean } = {
      id: new Date().getTime().toString(),
      action_id: action.id,
      configuration: {},
      uid: `${action.id}_${workflow.steps.length + 1}`,
      name: `${action.name} ${workflow.steps.length + 1}`,
      conditional: { language: "simple", expression: "true" },
      on_success: null,
      enabled: true,
      isNew: true,
    };

    setEditingStep(newStep);
  };

  useEffect(() => {
    const fetchWorkflowData = async function fetchWorkflowData() {
      const data = await getWorkflowById(id);
      setWorkflow(data);
    };
    const fetchRegistry = async function fetchRegistry() {
      return Promise.all([getAvailableActions(), getAvailableTriggers()]).then(
        ([actions, triggers]) => {
          setActionsRegistry(actions);
          setTriggerRegistry(triggers);
          console.log("actions", actions);
          console.log("triggers", triggers);
        }
      );
    };

    setIsLoading(true);
    setError(null);

    Promise.all([fetchWorkflowData(), fetchRegistry()])
      .then(() => {
        setIsLoading(false);
      })
      .catch((err) => {
        const apiError = err as ApiErrorProblem;
        console.error(`Failed to fetch workflow ${id}:`, apiError);
        setError(apiError.detail || "Failed to load workflow.");
      });
  }, [id]);

  useEffect(() => {
    if (!workflow) return;

    const triggerNodes: Partial<Node>[] = workflow.workflow_triggers.map<
      Partial<Node>
    >((trigger: TriggerItem) => {
      return {
        id: trigger.id,
        type: "trigger",
        data: { ...trigger },
      };
    });

    const stepsNodes: Partial<Node>[] = workflow.steps.map<Partial<Node>>(
      (step: WorkflowStep) => {
        return {
          id: step.id,
          type: "step",
          data: { ...step },
        };
      }
    );

    const stepEdges: Edge[] = workflow.steps.flatMap<Edge>(
      (step: WorkflowStep, index: number) => {
        const sEdges: Edge[] = [];

        const defaultEdge = {
          source: step.id,
        };

        if (index === 0) {
          workflow.workflow_triggers.forEach((trigger: TriggerItem) => {
            sEdges.push({
              ...defaultEdge,
              id: `trigger-${trigger.id}-to-${step.id}`,
              source: trigger.id,
              label: "starts",
              target: step.id,
              style: { stroke: "hsl(var(--accent))", strokeWidth: 1 },
            });
          });
        }

        if (step.on_failure) {
          sEdges.push(
            withDefaultParams(
              {
                source: step.id,
                id: `${step.id}-to-${step.on_failure}-failure`,
                target: step.on_failure,
              },
              false
            )
          );
        }

        if (step.on_success) {
          sEdges.push(
            withDefaultParams(
              {
                source: step.id,
                id: `${step.id}-to-${step.on_success}-success`,
                target: step.on_success,
              },
              true
            )
          );
        }

        return sEdges;
      }
    );

    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      [...triggerNodes, ...stepsNodes],
      stepEdges
    );

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
  }, [workflow, setNodes, setEdges]);

  return (
    <>
      {isLoading && (
        <div className="flex-grow flex flex-col items-center justify-center p-8 text-center">
          <Loader2 className="h-16 w-16 animate-spin text-primary" />
          <p className="ml-4 text-lg text-muted-foreground">
            Loading Workflow Editor...
          </p>
        </div>
      )}
      {error && !isLoading && (
        <div className="flex-grow flex flex-col items-center justify-center p-8 text-center">
          <AlertTriangle className="h-16 w-16 text-destructive mb-4" />
          <h2 className="text-2xl font-semibold text-destructive mb-2">
            Error Loading Workflow
          </h2>
          <p className="text-muted-foreground mb-6">{error}</p>
          <Button onClick={() => alert("go to /")} variant="outline">
            Go to Dashboard
          </Button>
        </div>
      )}
      {!error && !isLoading && (
        <>
          <aside className="w-1/4 min-w-[280px] max-w-[350px] flex flex-col h-auto space-y-4 p-4 overflow-y-auto bg-card">
            <h3 className="text-2xl font-semibold leading-none tracking-tight">
              Registry
            </h3>
            <div>
              <Palette
                items={[...triggersRegistry, ...actionsRegistry]}
                onSelect={handleAddItem}
              />
            </div>
          </aside>

          <div className="flex-grow w-3/4 lg:border-red-600 flex flex-col overflow-hidden h-[calc(100vh-64px)]">
            <div className="flex-grow border rounded-lg shadow-sm bg-card w-full">
              <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                nodeTypes={nodeTypes}
                nodeOrigin={[0.5, 0.5]}
                // TODO get fitview playing nice with nodeOrigin
                defaultViewport={{ x: 500, y: 0, zoom: 1 }}
                defaultEdgeOptions={{
                  animated: true,
                  markerEnd: {
                    type: MarkerType.ArrowClosed,
                    width: 20,
                    height: 20,
                    color: "var(--color-green-500)",
                  },
                }}
                fitView
              >
                <Background
                  variant={BackgroundVariant.Dots}
                  gap={12}
                  size={1}
                  className="bg-amber-100"
                />
                {/* <Controls /> */}

                {/* <DevTools /> */}
                <MiniMap />
              </ReactFlow>
            </div>
          </div>

          {editingStep && (
            <StepOrTriggerDialog
              item={editingStep}
              registry={actionsRegistry}
              type="step"
              onClose={() => setEditingStep(null)}
              onSaveItem={() => {}}
              currentWorkflow={workflow}
            />
          )}
        </>
      )}
    </>
  );
}
