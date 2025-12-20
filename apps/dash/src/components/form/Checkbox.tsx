import type { ComponentProps } from "react";

import { Checkbox, CheckedState } from "../ui/checkbox";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "../ui/field";
import { useFieldContext } from "./hook";

export function FormCheckbox({
  description,
  label,
  ...props
}: Omit<ComponentProps<typeof Checkbox>, "id" | "name"> & {
  description?: string;
  label: string;
}) {
  const field = useFieldContext<CheckedState>();
  const isInvalid = field.state.meta.isTouched && !field.state.meta.isValid;

  return (
    <Field data-invalid={isInvalid} orientation="horizontal">
      <Checkbox
        {...props}
        aria-invalid={isInvalid}
        checked={field.state.value}
        id={field.name}
        name={field.name}
        onCheckedChange={(checked) => field.handleChange(checked)}
      />
      <FieldContent>
        <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
        {description && <FieldDescription>{description}</FieldDescription>}
      </FieldContent>
      {isInvalid && <FieldError errors={field.state.meta.errors} />}
    </Field>
  );
}
