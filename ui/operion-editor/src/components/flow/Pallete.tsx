"use client";

import type { ActionRegistryItem, TriggerRegistryItem } from "@/types/operion";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/Card";
import { ScrollArea } from "@/components/ui/ScrollArea";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/Tooltip";
import { theme } from "./StepNode";
import { Badge } from "../ui/Badge";

interface PaletteProps {
  items: ActionRegistryItem[] | TriggerRegistryItem[];
  onSelect: (id: string) => void;
}

export function Palette({ items: items, onSelect }: PaletteProps) {
  return (
    <Card className="flex flex-col shadow-lg">
      <CardHeader>
        <CardDescription>
          Click to add an item to your workflow.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-grow p-0 overflow-hidden">
        <ScrollArea className="h-full p-4">
          <div className="space-y-3">
            {items.length === 0 && (
              <p className="text-sm text-muted-foreground">
                No items available.
              </p>
            )}
            {items.map((item) => {
              const Icon =
                theme[item.id as keyof typeof theme]?.icon ||
                theme.default.icon;

              return (
                <TooltipProvider key={item.id} delayDuration={200}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        onClick={() => onSelect(item.id)}
                        className="w-full text-left p-3 border rounded-lg hover:bg-blue-200 focus:outline-none transition-all duration-150 ease-in-out shadow-sm flex items-center space-x-3"
                        aria-label={`Add ${item.name}`}
                      >
                        <Icon className="h-5 w-5 text-primary shrink-0" />
                        <div className="flex-grow">
                          <p className="font-medium text-sm">{item.name}</p>
                          <p className="text-xs text-muted-foreground line-clamp-1">
                            {item.description}
                          </p>
                        </div>
                        <Badge variant={"outline"}>{item.type}</Badge>
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" className="max-w-xs">
                      <p className="font-semibold">
                        {item.name} ({item.id})
                      </p>
                      <p className="text-sm">{item.description}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              );
            })}
            <div> </div>
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
