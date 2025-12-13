import { APIError } from "@/lib/api";

import { withForm } from "./hook";

export const Form = withForm({
  props: {
    className: "",
  } as {
    className?: string;
  },
  render: function Form({ children, form, ...props }) {
    return (
      <form.AppForm>
        <form
          {...props}
          onSubmit={async (e) => {
            e.preventDefault();
            try {
              await form.handleSubmit();
            } catch (err) {
              if (err instanceof APIError) {
                const error = err.error;

                let formError: APIError["error"]["errors"][number] | undefined;
                const fieldsError: Record<
                  string,
                  APIError["error"]["errors"][number]
                > = {};

                for (const e of error.errors) {
                  if (e.location) {
                    fieldsError[e.location] = e;
                  }
                }

                if (
                  error.message !== error.errors[0]?.message ||
                  !error.errors[0]?.location
                ) {
                  formError = error;
                }

                form.setErrorMap({
                  onSubmit: {
                    fields: fieldsError,
                    form: formError,
                  },
                });

                console.error(err);

                return;
              }
              throw err;
            }
          }}
        >
          {children}
        </form>
      </form.AppForm>
    );
  },
});
