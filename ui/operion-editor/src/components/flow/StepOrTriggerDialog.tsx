import type {
  ActionRegistryItem,
  Workflow,
  WorkflowStep,
} from "@/types/operion";
import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  //   DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/Dialog";
import { ScrollArea } from "../ui/ScrollArea";
import { Button } from "../ui/Button";
import { Label } from "../ui/Label";

import { theme } from "./StepNode";
import { Input } from "../ui/Input";
import { DynamicForm } from "./DynamicFrom";

export default function StepOrTriggerDialog({
  item,
  type,
  registry,
  onClose,
}: // onSaveItem,
// currentWorkflow,
{
  item: WorkflowStep & { isNew?: boolean };
  // | (TriggerItem & { isNew?: boolean })
  type: "step";
  registry: ActionRegistryItem[];
  onClose: () => void;
  onSaveItem: (data: unknown) => void;
  currentWorkflow: Workflow | null;
}) {
  const [workingItem, setWorkingItem] = useState(item);
  const itemOnRegistry = registry.find((r) => r.id === item.action_id);

  const setName = (inputedName: string) => {
    if (workingItem) {
      const name = inputedName;
      setWorkingItem({
        ...workingItem,
        uid: name.toLowerCase().replace(/[^\p{L}\d]+/gu, "-"),
        name,
      });
    }
  };

  //   const [name, setName] = useState(item && "name" in item ? item.name : "");
  //   const [actionName, setActionName] = useState(
  //     item && "action" in item ? item.action.name : ""
  //   );
  //   const [actionDesc, setActionDesc] = useState(
  //     item && "action" in item ? item.action.description : ""
  //   );
  //   const [enabled, setEnabled] = useState(
  //     item && "enabled" in item ? item.enabled : true
  //   );
  //   const [configData, setConfigData] = useState(
  //     item
  //       ? "configuration" in item
  //         ? item.configuration || {}
  //         : ("action" in item && item.action.configuration) || {}
  //       : {}
  //   );
  //   const [onSuccess, setOnSuccess] = useState(
  //     item && "on_success" in item ? item.on_success || "" : ""
  //   );
  //   const [onFailure, setOnFailure] = useState(
  //     item && "on_failure" in item ? item.on_failure || "" : ""
  //   );

  useEffect(() => {
    // if (item) {
    //   setName(item && "name" in item ? item.name : "");
    //   setActionName(item && "action" in item ? item.action.name : "");
    //   setActionDesc(item && "action" in item ? item.action.description : "");
    //   setEnabled(item && "enabled" in item ? item.enabled : true);
    //   setConfigData(
    //     "configuration" in item
    //       ? item.configuration || {}
    //       : ("action" in item && item.action.configuration) || {}
    //   );
    //   setOnSuccess(item && "on_success" in item ? item.on_success || "" : "");
    //   setOnFailure(item && "on_failure" in item ? item.on_failure || "" : "");
    // }
  }, [item]);

  //   const itemSchema =
  //     type === "step" && item && "action" in item
  //       ? actionsRegistry.find((a) => a.type === item.action.type)
  //       : type === "trigger" && item
  //       ? triggersRegistry.find((t) => t.type === item.type)
  //       : undefined;

  const handleSubmit = () => {
    // let dataToSave;
    // if (type === "step" && item && "action" in item) {
    //   dataToSave = {
    //     name,
    //     action: {
    //       type: item.action.type,
    //       name: actionName,
    //       description: actionDesc,
    //       configuration: configData,
    //     },
    //     enabled,
    //     on_success:
    //       onSuccess === "" || onSuccess === NONE_SELECT_VALUE
    //         ? null
    //         : onSuccess,
    //     on_failure:
    //       onFailure === "" || onFailure === NONE_SELECT_VALUE
    //         ? null
    //         : onFailure,
    //   };
    // } else if (type === "trigger" && item) {
    //   dataToSave = {
    //     type: item.type,
    //     configuration: configData,
    //   };
    // }
    // if (dataToSave) onSaveItem(dataToSave);
  };

  if (!workingItem) return null;
  const titleDialog = workingItem.isNew ? `Add new ${type}` : `Edit ${type}`;
  const colorScheme =
    theme[workingItem.action_id as keyof typeof theme] || theme.default;

  const Icon = colorScheme.icon;
  //   const Icon =
  //     WorkflowIconMapping[
  //       type === "step" && "action" in item
  //         ? item.action.type
  //         : type === "trigger" && "type" in item
  //         ? item.type
  //         : "default"
  //     ] || HelpCircle;

  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2 font-headline">
            {<Icon className="h-5 w-5 text-primary" />}
            <span>
              {titleDialog}: {workingItem.uid}
            </span>
          </DialogTitle>
        </DialogHeader>
        <ScrollArea className="max-h-[70vh] p-1 pr-4">
          {type === "step" && (
            <div className="space-y-4 py-4">
              <div>
                <Label htmlFor="itemName">Name</Label>
                <Input
                  id="itemName"
                  value={workingItem.name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="My step name"
                />
              </div>
            </div>
          )}

          <Label>Configuration</Label>
          {itemOnRegistry?.schema &&
          Object.keys(itemOnRegistry.schema).length > 0 ? (
            <DynamicForm
              schema={itemOnRegistry.schema}
              defaultValues={item.configuration || {}}
              onChange={(configuration) => {
                setWorkingItem({
                  ...workingItem,
                  configuration,
                });
              }}
              formId={`${type}-${item?.id}-config-subform`}
            />
          ) : (
            <p className="text-sm text-muted-foreground">
              This {type} has no configurable options.
            </p>
          )}

          {/*  <div className="space-y-4 py-4">


            {type === "step" && currentWorkflow && (
              <>
                <Separator className="my-4" />
                <h4 className="text-md font-semibold">Transitions</h4>
                <div>
                  <Label htmlFor="itemOnSuccess">On Success (Next Step)</Label>
                  <Select
                    value={onSuccess || NONE_SELECT_VALUE}
                    onValueChange={(value) => setOnSuccess(value)}
                  >
                    <SelectTrigger id="itemOnSuccess">
                      <SelectValue placeholder="Select next step on success" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={NONE_SELECT_VALUE}>
                        -- None --
                      </SelectItem>
                      {currentWorkflow.steps
                        .filter((s) => item && "id" in item && s.id !== item.id)
                        .map((s) => (
                          <SelectItem key={s.id} value={s.id}>
                            {s.name} ({s.id.substring(0, 5)}...)
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="itemOnFailure">On Failure (Next Step)</Label>
                  <Select
                    value={onFailure || NONE_SELECT_VALUE}
                    onValueChange={(value) => setOnFailure(value)}
                  >
                    <SelectTrigger id="itemOnFailure">
                      <SelectValue placeholder="Select next step on failure" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={NONE_SELECT_VALUE}>
                        -- None --
                      </SelectItem>
                      {currentWorkflow.steps
                        .filter((s) => item && "id" in item && s.id !== item.id)
                        .map((s) => (
                          <SelectItem key={s.id} value={s.id}>
                            {s.name} ({s.id.substring(0, 5)}...)
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
              </>
            )}

            {type === "step" && "enabled" in item && (
              <div className="flex items-center space-x-2 pt-4">
                <Switch
                  id="itemEnabled"
                  checked={enabled}
                  onCheckedChange={setEnabled}
                />
                <Label htmlFor="itemEnabled">Enabled</Label>
              </div>
            )}
          </div>
          */}
        </ScrollArea>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            type="submit"
            form={`${type}-${item?.id}-config-subform`}
          >
            {item.isNew ? "Add" : "Save"} {type}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
