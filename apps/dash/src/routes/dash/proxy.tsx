import { createFileRoute } from "@tanstack/react-router";
import { CopyIcon, LinkIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import z from "zod";

import { useProxifyLinkMutation } from "@/api/proxy";
import { Form, useAppForm } from "@/components/form";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export const Route = createFileRoute("/dash/proxy")({
  component: RouteComponent,
  staticData: {
    crumb: "Proxy",
  },
});

const formSchema = z.object({
  encrypt: z.boolean(),
  exp: z.string().optional(),
  filename: z.string().optional(),
  req_headers: z.string().optional(),
  url: z.url(),
});

function RouteComponent() {
  const [proxyUrl, setProxyUrl] = useState<string>("");

  const proxify = useProxifyLinkMutation();

  const form = useAppForm({
    defaultValues: {
      encrypt: true,
      exp: "",
      filename: "",
      req_headers: "",
      url: "",
    } as z.infer<typeof formSchema>,
    onSubmit: async ({ value }) => {
      setProxyUrl("");
      const result = await proxify.mutateAsync(value);
      setProxyUrl(result.data.url);
    },
    validators: {
      onChange: formSchema,
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Generate Proxy Link</CardTitle>
          <CardDescription>
            Create a proxified URL that routes through StremThru
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form className="flex flex-col gap-4" form={form}>
            <form.AppField name="url">
              {(field) => (
                <field.Input
                  label="URL"
                  placeholder="https://emojiapi.dev/api/v1/sparkles/256.png"
                  required
                />
              )}
            </form.AppField>

            <div className="flex flex-row justify-between gap-4">
              <div className="max-w-96">
                <form.AppField name="exp">
                  {(field) => (
                    <field.Input
                      label="Expiration"
                      placeholder="e.g. 1h, 30m, 24h"
                    />
                  )}
                </form.AppField>
              </div>

              <div className="flex-1">
                <form.AppField name="filename">
                  {(field) => (
                    <field.Input
                      label="Filename"
                      placeholder="sparkles-256.png"
                    />
                  )}
                </form.AppField>
              </div>
            </div>

            <form.AppField name="req_headers">
              {(field) => (
                <field.Textarea
                  label="Request Headers"
                  placeholder={"Header-Name: value\n(one per line)"}
                />
              )}
            </form.AppField>

            <form.AppField name="encrypt">
              {(field) => (
                <field.Checkbox
                  description="Encrypt the proxified link"
                  label="Encrypt"
                />
              )}
            </form.AppField>

            <div>
              <form.AppForm>
                <form.SubmitButton>
                  <LinkIcon className="size-4" />
                  Generate Proxy Link
                </form.SubmitButton>
              </form.AppForm>
            </div>
          </Form>
        </CardContent>
      </Card>

      {proxyUrl && (
        <Card>
          <CardHeader>
            <CardTitle>Proxy Link</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Input
                className="font-mono text-sm"
                onClick={(e) => (e.target as HTMLInputElement).select()}
                readOnly
                value={proxyUrl}
              />
              <Button
                onClick={() => {
                  navigator.clipboard.writeText(proxyUrl);
                  toast.success("Copied to clipboard");
                }}
                size="icon"
                type="button"
                variant="outline"
              >
                <CopyIcon className="size-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
