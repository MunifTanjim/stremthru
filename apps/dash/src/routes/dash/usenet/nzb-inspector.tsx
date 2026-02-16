import { createFileRoute } from "@tanstack/react-router";
import { FileText, SaveIcon, UploadIcon, XIcon } from "lucide-react";
import { DateTime } from "luxon";
import prettyBytes from "pretty-bytes";
import { useState } from "react";
import { toast } from "sonner";
import z from "zod";

import {
  ParsedNZB,
  useNzbParseMutation,
  useNzbUploadMutation,
} from "@/api/usenet";
import { Form, useAppForm } from "@/components/form";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  FileUploadDropzone,
  FileUploadItem,
  FileUploadItemDelete,
  FileUploadItemMetadata,
  FileUploadItemPreview,
  FileUploadList,
  FileUploadTrigger,
} from "@/components/ui/file-upload";
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";

export const Route = createFileRoute("/dash/usenet/nzb-inspector")({
  component: RouteComponent,
  staticData: {
    crumb: "NZB",
  },
});

function age(dateString: string): null | string {
  return DateTime.fromISO(dateString)
    .diffNow()
    .negate()
    .shiftTo("years", "months", "days", "hours")
    .removeZeros()
    .toHuman({
      maximumFractionDigits: 0,
      unitDisplay: "short",
    });
}

function formatDate(dateString: string): string {
  return DateTime.fromISO(dateString).toLocaleString(DateTime.DATETIME_MED);
}

const formSchema = z.object({
  file: z
    .file()
    .max(5 * 1024 * 1024)
    .nullable(),
});

function RouteComponent() {
  const [parsedNzb, setParsedNzb] = useState<null | ParsedNZB>(null);

  const parse = useNzbParseMutation();
  const upload = useNzbUploadMutation();

  const form = useAppForm({
    defaultValues: { file: null } as z.infer<typeof formSchema>,
    onSubmit: async ({ value }) => {
      setParsedNzb(null);

      if (!value.file) {
        return;
      }

      toast.promise(parse.mutateAsync(value.file), {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Parsing NZB file...",
        success(data) {
          setParsedNzb(data);
          return {
            closeButton: true,
            message: "NZB parsed successfully!",
          };
        },
      });
    },
    validators: {
      onChange: formSchema,
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Parse NZB File</CardTitle>
          <CardDescription>
            Upload an NZB file to see its contents and metadata
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form className="flex flex-col gap-4" form={form}>
            <form.AppField name="file">
              {(field) => (
                <field.FilePicker accept=".nzb" maxFiles={1}>
                  {field.state.value ? (
                    <FileUploadList>
                      <FileUploadItem value={field.state.value}>
                        <FileUploadItemPreview />
                        <FileUploadItemMetadata />
                        <FileUploadItemDelete asChild>
                          <Button
                            aria-label="Remove file"
                            className="size-7"
                            size="icon"
                            variant="ghost"
                          >
                            <XIcon />
                          </Button>
                        </FileUploadItemDelete>
                      </FileUploadItem>
                    </FileUploadList>
                  ) : (
                    <FileUploadDropzone className="p-4">
                      <div className="flex flex-row items-center gap-2 text-center">
                        <div className="flex items-center justify-center rounded-full border p-2">
                          <UploadIcon className="text-muted-foreground size-6" />
                        </div>
                        Drag & drop NZB file here or
                        <FileUploadTrigger asChild>
                          <Button size="sm" variant="outline">
                            Browse
                          </Button>
                        </FileUploadTrigger>
                      </div>
                    </FileUploadDropzone>
                  )}
                </field.FilePicker>
              )}
            </form.AppField>

            <div className="flex gap-2">
              <Button
                className="w-fit"
                disabled={parse.isPending}
                type="submit"
              >
                {parse.isPending ? (
                  <Spinner />
                ) : (
                  <FileText className="size-4" />
                )}
                Parse NZB
              </Button>
              {parsedNzb && (
                <form.Subscribe selector={(s) => s.values.file}>
                  {(file) =>
                    file ? (
                      <Button
                        className="w-fit"
                        disabled={upload.isPending}
                        onClick={(e) => {
                          e.preventDefault();
                          const name = parsedNzb.meta.title || file.name;
                          toast.promise(upload.mutateAsync({ file, name }), {
                            error(err: APIError) {
                              console.error(err);
                              return {
                                closeButton: true,
                                message: err.message,
                              };
                            },
                            loading: "Uploading NZB file...",
                            success() {
                              return {
                                closeButton: true,
                                message: "NZB queued for processing!",
                              };
                            },
                          });
                        }}
                        type="button"
                        variant="secondary"
                      >
                        {upload.isPending ? (
                          <Spinner />
                        ) : (
                          <SaveIcon className="size-4" />
                        )}
                        Save
                      </Button>
                    ) : null
                  }
                </form.Subscribe>
              )}
            </div>
          </Form>
        </CardContent>
      </Card>

      {parsedNzb && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>NZB Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <div className="text-muted-foreground font-medium">
                    Total Size
                  </div>
                  <div className="mt-1">{prettyBytes(parsedNzb.size)}</div>
                </div>
                <div>
                  <div className="text-muted-foreground font-medium">
                    File Count
                  </div>
                  <div className="mt-1">{parsedNzb.files.length}</div>
                </div>
                {Object.entries(parsedNzb.meta).map(([key, value]) => (
                  <div key={key}>
                    <div className="text-muted-foreground font-medium capitalize">
                      {key}
                    </div>
                    <div className="mt-1">{value}</div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Files</CardTitle>
              <CardDescription>
                {parsedNzb.files.length} file
                {parsedNzb.files.length > 1 ? "s" : ""} in this NZB
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {parsedNzb.files.map((file) => (
                  <div
                    className="border-border rounded-lg border p-4"
                    key={file.subject}
                  >
                    <div className="flex flex-wrap items-start justify-between gap-4">
                      <div className="flex-1 space-y-1">
                        <h3 className="break-all font-medium">{file.name}</h3>
                        {file.name !== file.subject && (
                          <p className="text-muted-foreground text-sm">
                            Subject: {file.subject}
                          </p>
                        )}
                        <p className="text-muted-foreground text-sm">
                          Posted by: {file.poster}
                        </p>
                      </div>
                      <div className="ml-auto text-right text-sm">
                        <div className="font-medium">
                          {prettyBytes(file.size)}
                        </div>
                        <div className="text-muted-foreground">
                          {file.segments.length} segment
                          {file.segments.length > 1 ? "s" : ""}
                        </div>
                        {file.date && (
                          <Tooltip>
                            <TooltipTrigger>
                              <div className="text-muted-foreground">
                                {age(file.date)}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent side="left">
                              {formatDate(file.date)}
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </div>
                    </div>
                    {file.groups.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {file.groups.map((group) => (
                          <span
                            className="bg-muted text-muted-foreground rounded px-2 py-1 text-xs"
                            key={group}
                          >
                            {group}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
