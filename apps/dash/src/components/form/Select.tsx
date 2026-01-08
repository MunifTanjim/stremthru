import type { ComponentProps } from "react";

import { XIcon } from "lucide-react";

import { Button } from "../ui/button";
import { Field, FieldError, FieldLabel } from "../ui/field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import { useFieldContext } from "./hook";

export function FormSelect({
  label,
  options,
  placeholder,
  ...props
}: Omit<ComponentProps<typeof Select>, "id" | "name"> & {
  label: string;
  options: Array<{ label: string; value: string }>;
  placeholder?: string;
}) {
  const field = useFieldContext<string>();
  const isInvalid = field.state.meta.isTouched && !field.state.meta.isValid;

  let onReset: ComponentProps<typeof Button>["onClick"];
  if (!props.required && field.state.value) {
    onReset = () => field.setValue("");
  }

  return (
    <Field data-invalid={isInvalid}>
      <FieldLabel htmlFor={field.name}>
        {label}

        {onReset && (
          <Button
            className="hover:bg-inherit! p-0! ml-auto mr-3 h-auto"
            onClick={onReset}
            variant="ghost"
          >
            <XIcon />
          </Button>
        )}
      </FieldLabel>
      <Select
        {...props}
        onValueChange={(value) => field.handleChange(value)}
        value={field.state.value}
      >
        <SelectTrigger aria-invalid={isInvalid} id={field.name}>
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {isInvalid && <FieldError errors={field.state.meta.errors} />}
    </Field>
  );
}
