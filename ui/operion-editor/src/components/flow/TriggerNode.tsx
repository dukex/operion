import { Handle, Position } from "@xyflow/react";
import { memo } from "react";
import type { TriggerItem } from "../../types/operion";
import { SatelliteDish } from "lucide-react";
import BaseNode from "./BaseNode";

const TriggerNode = memo(
  ({ data, isConnectable }: { data: TriggerItem; isConnectable: boolean }) => {
    return (
      <>
        <BaseNode
          color="var(--color-green-700)"
          icon={<SatelliteDish />}
          background="var(--color-green-100)"
          title="Trigger Node"
        >
          {data.trigger_id == "schedule" && (
            <div>
              Schedule:{" "}
              <span className="font-normal">{data.configuration.schedule}</span>
            </div>
          )}

          <Handle
            type="source"
            position={Position.Bottom}
            isConnectable={isConnectable}
            style={{
              background: "var(--color-green-700)",
              width: 10,
              height: 10,
            }}
          />
        </BaseNode>
      </>
    );
  }
);

export default TriggerNode;
