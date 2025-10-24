import { createFileRoute, Navigate } from "@tanstack/react-router";
import { Sparkles } from "lucide-react";
import { z } from "zod";

import { useSignIn } from "@/api/auth";
import { Form, useAppForm } from "@/components/form";
import { FieldGroup } from "@/components/ui/field";
import { useCurrentAuth } from "@/hooks/auth";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/dash/login")({
  component: RouteComponent,
});

function LoginForm({ className, ...props }: React.ComponentProps<"div">) {
  const signIn = useSignIn("password");

  const form = useAppForm({
    defaultValues: {
      password: "",
      user: "",
    },
    onSubmit: async ({ value }) => {
      await signIn.mutateAsync(value);
    },
    validators: {
      onChange: z.object({
        password: z.string().nonempty(),
        user: z.string().nonempty(),
      }),
    },
  });

  return (
    <Form
      {...props}
      className={cn("flex flex-col gap-6", className)}
      form={form}
    >
      <FieldGroup>
        <div className="flex flex-col items-center gap-2 text-center">
          <a className="flex flex-col items-center gap-2 font-medium" href="#">
            <div className="flex size-8 items-center justify-center rounded-md">
              <Sparkles className="size-6" />
            </div>
            <span className="sr-only">StremThru</span>
          </a>
          <h1 className="text-xl font-bold">Welcome to StremThru!</h1>
        </div>

        <form.AppField name="user">
          {(field) => <field.Input label="User" required type="text" />}
        </form.AppField>

        <form.AppField name="password">
          {(field) => <field.Input label="Password" required type="password" />}
        </form.AppField>

        <form.SubmitButton>Login</form.SubmitButton>
      </FieldGroup>
    </Form>
  );
}

function RouteComponent() {
  const { user } = useCurrentAuth();
  if (user) {
    return <Navigate to="/dash" />;
  }

  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <LoginForm />
      </div>
    </div>
  );
}
