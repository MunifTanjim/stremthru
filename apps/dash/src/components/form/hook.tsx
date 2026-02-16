import { createFormHook, createFormHookContexts } from "@tanstack/react-form";

import { FormCheckbox } from "./Checkbox";
import { FormFilePicker } from "./File";
import { FormInput } from "./Input";
import { FormSelect } from "./Select";
import { SubmitButton } from "./SubmitButton";
import { FormTextarea } from "./Textarea";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
  fieldComponents: {
    Checkbox: FormCheckbox,
    FilePicker: FormFilePicker,
    Input: FormInput,
    Select: FormSelect,
    Textarea: FormTextarea,
  },
  fieldContext,
  formComponents: {
    SubmitButton,
  },
  formContext,
});
