import { useParams } from "react-router";
import type { Workflow, ApiErrorProblem } from "@/types/operion";
import { useCallback, useEffect, useState } from "react";
import { getWorkflowById } from "@/lib/api-client";
import { AlertTriangle } from "lucide-react";
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

const dagreGraph = new dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));

export default function WorkflowsGet() {
  const params = useParams();
  const id = params.id as string;

  const [workflow, setWorkflow] = useState<Workflow | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const onConnect = useCallback(
    (params: Edge | Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  useEffect(() => {
    const fetchWorkflowData = async function fetchWorkflowData() {
      setIsLoading(true);
      setError(null);
      try {
        const data = await getWorkflowById(id);
        setWorkflow(data);
        //   setSelectedItem({ type: "workflow", data });
        //   setExplicitEntryStepId(null); // Reset on new workflow load
      } catch (err) {
        const apiError = err as ApiErrorProblem;
        console.error(`Failed to fetch workflow ${id}:`, apiError);
        setError(apiError.detail || "Failed to load workflow.");
      } finally {
        setIsLoading(false);
      }
    };

    fetchWorkflowData();
    // fetchRegistry();
  }, [id]);

  useEffect(() => {
    if (!workflow) return;

    const triggerNodes: Partial<Node>[] = workflow.triggers.map<Partial<Node>>(
      (trigger: TriggerItem) => {
        return {
          id: trigger.id,
          type: "trigger",
          data: { ...trigger },
        };
      }
    );

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
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 20,
            height: 20,
            color: "var(--color-green-500)",
          },
        };

        if (index === 0) {
          workflow.triggers.forEach((trigger: TriggerItem) => {
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

        if (step.on_success) {
          sEdges.push({
            ...defaultEdge,
            id: `${step.id}-to-${step.on_success}-success`,
            sourceHandle: "success",
            label: "on success",
            target: step.on_success,
            style: { stroke: "var(--color-green-500)", strokeWidth: 1 },
          });
        }
        if (step.on_failure) {
          sEdges.push({
            ...defaultEdge,
            id: `${step.id}-to-${step.on_failure}-failure`,
            sourceHandle: "failure",
            label: "on failure",
            target: step.on_failure,
            style: { stroke: "hsl(var(--destructive))", strokeWidth: 1 },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              width: 20,
              height: 20,
              color: "hsl(var(--destructive))",
            },
          });
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
        <div className="flex-grow w-1/2 xl:w-3/5 flex flex-col overflow-hidden h-[calc(100vh-64px)]">
          <div className="flex-grow border rounded-lg shadow-sm bg-card w-full">
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              nodeTypes={nodeTypes}
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
      )}
    </>
  );
}
