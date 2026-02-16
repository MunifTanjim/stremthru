import { type ComponentProps, useMemo } from "react";
import { ZodFile, ZodNullable, ZodObject, ZodOptional } from "zod";

import { Field } from "../ui/field";
import { FileUpload } from "../ui/file-upload";
import { useFieldContext } from "./hook";

export const FormFilePicker = Object.assign(function FormFile({
  ...props
}: Omit<
  ComponentProps<typeof FileUpload>,
  | "aria-invalid"
  | "id"
  | "maxFiles"
  | "name"
  | "onBlur"
  | "onValueChange"
  | "value"
>) {
  const field = useFieldContext<File>();
  const isInvalid = field.state.meta.isTouched && !field.state.meta.isValid;

  const schema = useMemo(() => {
    const schema = field.form.options.validators?.onChange;
    if (!schema) {
      return null;
    }
    if (schema instanceof ZodObject) {
      const s = schema.shape[field.name];
      if (s instanceof ZodFile) {
        return s;
      }
      if (s instanceof ZodNullable || s instanceof ZodOptional) {
        const f = s.unwrap();
        if (f instanceof ZodFile) {
          return f;
        }
      }
      return null;
    }
    return null;
  }, [field.form.options.validators?.onChange, field.name]);

  return (
    <Field data-invalid={isInvalid}>
      <FileUpload
        {...props}
        aria-invalid={isInvalid}
        id={field.name}
        invalid={isInvalid}
        maxFiles={1}
        name={field.name}
        onBlur={field.handleBlur}
        onFileReject={(_, message) => {
          field.setErrorMap({
            onChange: message,
          });
        }}
        onFileValidate={(f) => {
          if (!schema) {
            return;
          }
          const r = schema.safeParse(f);
          if (r.success) {
            return;
          }
          return r.error.message;
        }}
        onValueChange={(files) => {
          field.handleChange(files[0]);
        }}
        value={[field.state.value].filter(Boolean)}
      />
    </Field>
  );
});
