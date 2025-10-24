import { useStore } from "@tanstack/react-form";
import { ComponentProps } from "react";

import { Button } from "../ui/button";
import { Field, FieldError } from "../ui/field";
import { Spinner } from "../ui/spinner";
import { useFormContext } from "./hook";

export function SubmitButton({
  children,
  ...props
}: ComponentProps<typeof Button>) {
  const form = useFormContext();
  const errors = useStore(form.store, (state) => state.errors);
  const canSubmit = useStore(form.store, (state) => state.canSubmit);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  return (
    <form.Subscribe>
      <Field>
        <FieldError errors={errors} />
        <Button {...props} disabled={!canSubmit} type="submit">
          {isSubmitting && <Spinner />}
          {children}
        </Button>
      </Field>
    </form.Subscribe>
  );
}
