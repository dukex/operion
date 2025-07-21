import { useEffect } from "react";
import {
  useForm,
  Controller,
  type Control,
  type ControllerRenderProps,
  type FieldValues,
} from "react-hook-form";
import type { JsonSchemaMinimal } from "@/types/operion";
import { Input } from "@/components/ui/Input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select";
import { Label } from "@/components/ui/Label";
import { Textarea } from "@/components/ui/Textarea";
import { Card, CardContent } from "@/components/ui/Card";

interface DynamicFormProps {
  schema: JsonSchemaMinimal;
  onChange: (data: Record<string, unknown>) => void;
  formId?: string;
  defaultValues?: Record<string, unknown>;
}

const renderSimpleFields = ({
  name,
  schema,
  fullPath = "",
  required = false,
  control,
}: {
  name: string;
  schema: JsonSchemaMinimal & { type: "string" | "number" | "integer" };
  fullPath: string;
  required?: boolean;
  control: Control;
}) => {
  const render = ({
    field,
  }: {
    field: ControllerRenderProps<FieldValues, string>;
  }) => {
    switch (schema.type) {
      case "string":
        if (schema.enum && schema.enum.length > 0) {
          return (
            <Select
              onValueChange={field.onChange}
              value={field.value as string}
            >
              <SelectTrigger>
                <SelectValue placeholder={`Select ${name}`} />
              </SelectTrigger>
              <SelectContent>
                {schema.enum!.map((option) => (
                  <SelectItem key={option} value={option}>
                    {option}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          );
        }
        return schema.format === "text" ? (
          <Textarea
            {...field}
            value={field.value as string | undefined}
            rows={3}
          />
        ) : schema.format === "code" ? (
          <Textarea
            {...field}
            value={field.value as string | undefined}
            rows={5}
            className="font-mono"
          />
        ) : (
          <Input
            {...field}
            value={field.value as string | undefined}
            type={schema.format || "text"}
          />
        );
      case "number":
      case "integer":
        return (
          <Input
            {...field}
            value={field.value as number | undefined}
            max={schema.maximum}
            min={schema.minimum}
            type="number"
            onChange={(e) => field.onChange(parseFloat(e.target.value))}
          />
        );
    }
  };

  const examples =
    schema.examples !== undefined && schema.examples.length > 0
      ? schema.examples
      : schema.example
      ? [schema.example]
      : [];
  const hasExamples = examples.length > 0;

  return (
    <div className="flex flex-row space-x-2">
      <div className={hasExamples ? "w-3/4" : "w-full"}>
        <Controller
          name={fullPath}
          control={control}
          rules={{ required: required }}
          defaultValue={schema.default}
          render={render}
        />
      </div>
      {hasExamples && (
        <div className="w-2/4">
          <p className="text-xs">Examples:</p>
          {examples.map((example) => (
            <div className="mb-2 text-xs font-mono border p-2">{example}</div>
          ))}
        </div>
      )}
    </div>
  );
};

const renderField = ({
  name,
  schema,
  parentPath = "",
  required = false,
  control,
}: {
  name: string;
  schema: JsonSchemaMinimal;
  parentPath?: string;
  required?: boolean;
  control: Control;
}) => {
  const fullPath =
    name === "root" ? "" : parentPath ? `${parentPath}.${name}` : name;

  switch (schema.type) {
    case "string":
    case "number":
    case "integer":
      return renderSimpleFields({
        name,
        schema: schema as JsonSchemaMinimal & {
          type: "string" | "number" | "integer";
        },
        fullPath,
        required,
        control,
      });
    case "object":
      return (
        <Card className="mt-2 p-4 border-dashed">
          <CardContent className="space-y-4 p-0">
            {Object.entries(schema.properties || {}).map(
              ([subFieldName, subFieldSchema]) => (
                <div key={subFieldName} className="space-y-2">
                  <div>
                    <Label
                      htmlFor={`${fullPath}.${subFieldName}`}
                      className="text-xs font-bold text-muted-foreground capitalize"
                    >
                      {subFieldSchema.title || subFieldName.replace(/_/g, " ")}
                      {schema.required?.find((f) => f === subFieldName) && "*"}
                    </Label>
                    <p className="text-xs font-medium text-muted-foreground">
                      {subFieldSchema.description}
                    </p>
                  </div>
                  {/* TODO: handle with additionalProperties, handle `headers` is a object
                        with additionalProperties	= { "type": "string" }, we should given the user
                        option to add many key,value string ites object here	 */}
                  {renderField({
                    name: subFieldName,
                    schema: subFieldSchema,
                    parentPath: fullPath,
                    required: !!schema.required?.find(
                      (f) => f === subFieldName
                    ),
                    control: control,
                  })}

                  {/* {errors[fullPath]?.[subFieldName] && (
                       <p className="text-xs text-destructive">
                         This field is required.
                       </p>
                     )} */}
                </div>
              )
            )}
          </CardContent>
        </Card>
      );
    default:
      return (
        <p className="text-sm text-destructive">
          Unsupported field type: {schema.type}
        </p>
      );
  }
};

export function DynamicForm({
  schema,
  onChange,
  formId,
  defaultValues,
}: DynamicFormProps) {
  const {
    control,
    reset,
    formState: { errors },
    watch,
  } = useForm({
    defaultValues: defaultValues || {},
  });

  useEffect(() => {
    reset();
  }, [reset]);

  useEffect(() => {
    const subscription = watch((data) => {
      onChange(data);
    });
    return () => subscription.unsubscribe();
  }, [watch, onChange]);

  console.log("errors", errors);

  return (
    <form id={formId} className="space-y-6">
      {renderField({ name: "root", schema, control })}
    </form>
  );
}
