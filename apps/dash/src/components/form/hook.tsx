import { createFormHook, createFormHookContexts } from "@tanstack/react-form";

import { FormFilePicker } from "./File";
import { FormInput } from "./Input";
import { FormSelect } from "./Select";
import { SubmitButton } from "./SubmitButton";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
  fieldComponents: {
    FilePicker: FormFilePicker,
    Input: FormInput,
    Select: FormSelect,
  },
  fieldContext,
  formComponents: {
    SubmitButton,
  },
  formContext,
});
