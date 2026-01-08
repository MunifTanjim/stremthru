import { createFormHook, createFormHookContexts } from "@tanstack/react-form";

import { FormInput } from "./Input";
import { FormSelect } from "./Select";
import { SubmitButton } from "./SubmitButton";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
  fieldComponents: {
    Input: FormInput,
    Select: FormSelect,
  },
  fieldContext,
  formComponents: {
    SubmitButton,
  },
  formContext,
});
