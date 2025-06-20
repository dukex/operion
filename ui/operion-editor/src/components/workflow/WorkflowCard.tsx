import type { Workflow } from "@/types/operion";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { ArrowRight, CalendarDays, Layers, UserCircle } from "lucide-react";
import { formatDistanceToNow } from "date-fns";

interface WorkflowCardProps {
  workflow: Workflow;
  //   onDelete: (id: string) => void;
}

export function WorkflowCard({ workflow }: WorkflowCardProps) {
  return (
    <Card className="flex flex-col h-full hover:shadow-lg transition-shadow duration-200">
      <CardHeader>
        <div className="flex justify-between items-start">
          <CardTitle className="text-xl font-headline">
            {workflow.name}
          </CardTitle>
          <Badge
            variant={workflow.status === "active" ? "default" : "secondary"}
            className="capitalize"
          >
            {workflow.status}
          </Badge>
        </div>
        <CardDescription className="line-clamp-2 h-[2.5em]">
          {workflow.description}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-grow space-y-3 text-sm text-muted-foreground">
        <div className="flex items-center">
          <Layers className="mr-2 h-4 w-4" />
          <span>
            {workflow.steps.length} step{workflow.steps.length !== 1 ? "s" : ""}
          </span>
        </div>
        <div className="flex items-center">
          <UserCircle className="mr-2 h-4 w-4" />
          <span>Owner: {workflow.owner || "N/A"}</span>
        </div>
        <div className="flex items-center">
          <CalendarDays className="mr-2 h-4 w-4" />
          <span>
            Updated{" "}
            {formatDistanceToNow(new Date(workflow.updated_at), {
              addSuffix: true,
            })}
          </span>
        </div>
      </CardContent>
      <CardFooter className="flex justify-between items-center">
        {/* <Button
          variant="destructive"
          size="sm"
          onClick={() => onDelete(workflow.id)}
        >
          Delete
        </Button> */}
        <Button asChild variant="outline" size="sm">
          <a href={`/workflows/${workflow.id}`}>
            Open Editor <ArrowRight className="ml-2 h-4 w-4" />
          </a>
        </Button>
      </CardFooter>
    </Card>
  );
}
