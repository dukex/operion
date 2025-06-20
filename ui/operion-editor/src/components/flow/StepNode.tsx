import { Handle, Position } from "@xyflow/react";
import { memo } from "react";
import type { WorkflowStep } from "../../types/operion";
import { BoxIcon, CloudCog, FilePenLine, SquareFunction } from "lucide-react";
import BaseNode from "./BaseNode";

const colors = {
  http_request: {
    color: "var(--color-blue-400)",
    background: "var(--color-blue-100)",
    icon: <CloudCog />,
  },
  transform: {
    color: "var(--color-orange-400)",
    background: "var(--color-orange-100)",
    icon: <SquareFunction />,
  },
  file_write: {
    color: "var(--color-yellow-500)",
    background: "var(--color-yellow-100)",
    icon: <FilePenLine />,
  },
  default: {
    color: "var(--color-gray-500)",
    background: "var(--color-gray-100)",
    icon: <BoxIcon />,
  },
};

const StepNode = memo(
  ({ data, isConnectable }: { data: WorkflowStep; isConnectable: boolean }) => {
    const colorScheme =
      colors[data.action.type as keyof typeof colors] || colors.default;

    return (
      <BaseNode
        color={colorScheme.color}
        background={colorScheme.background}
        icon={colorScheme.icon}
        title={data.action.type
          .split("_")
          .map((s) => (s === "http" ? "HTTP" : s))
          .join(" ")}
      >
        <>
          {data.name || "No name provided"}
          <Handle
            type="source"
            id="success"
            position={Position.Bottom}
            isConnectable={isConnectable}
            style={{
              left: "60%",
              background: "var(--color-green-500)",
              width: 10,
              height: 10,
            }}
          />
          <Handle
            type="source"
            id="failure"
            position={Position.Bottom}
            isConnectable={isConnectable}
            style={{
              left: "40%",
              background: "var(--color-red-500)",
              width: 10,
              height: 10,
            }}
          />
          <Handle
            type="target"
            position={Position.Top}
            isConnectable={isConnectable}
            style={{
              background: colorScheme.color,
              width: 10,
              height: 10,
            }}
          />
        </>
      </BaseNode>
    );
  }
);

export default StepNode;
