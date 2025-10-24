import { createFormHook, createFormHookContexts } from "@tanstack/react-form";

import { FormInput } from "./Input";
import { SubmitButton } from "./SubmitButton";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
  fieldComponents: {
    Input: FormInput,
  },
  fieldContext,
  formComponents: {
    SubmitButton,
  },
  formContext,
});
