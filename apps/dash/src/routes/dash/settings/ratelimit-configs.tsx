import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { DateTime } from "luxon";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import z from "zod";

import {
  RateLimitConfig,
  useRateLimitConfigMutation,
  useRateLimitConfigs,
} from "@/api/ratelimit-config";
import { DataTable } from "@/components/data-table";
import { useDataTable } from "@/components/data-table/use-data-table";
import { Form } from "@/components/form/Form";
import { useAppForm } from "@/components/form/hook";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    RateLimitConfig: {
      onEdit: (item: RateLimitConfig) => void;
      removeConfig: ReturnType<typeof useRateLimitConfigMutation>["remove"];
    };
  }

  export interface DataTableMetaCtxKey {
    RateLimitConfig: RateLimitConfig;
  }
}

const col = createColumnHelper<RateLimitConfig>();

const columns: ColumnDef<RateLimitConfig>[] = [
  col.accessor("name", {
    header: "Name",
  }),
  col.accessor("limit", {
    header: "Limit",
  }),
  col.accessor("window", {
    header: "Window",
  }),
  col.accessor("updated_at", {
    cell: ({ getValue }) => {
      const date = DateTime.fromISO(getValue());
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Updated At",
  }),
  col.display({
    cell: (c) => {
      const { onEdit, removeConfig } = c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                onClick={() => onEdit(item)}
                size="icon-sm"
                variant="ghost"
              >
                <Pencil />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Edit</TooltipContent>
          </Tooltip>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button size="icon-sm" variant="ghost">
                <Trash2 className="text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Rate Limit Config?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the rate limit configuration{" "}
                  <strong>{item.name}</strong>. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={removeConfig.isPending}
                    onClick={() => {
                      toast.promise(removeConfig.mutateAsync(item.id), {
                        error(err: APIError) {
                          console.error(err);
                          return {
                            closeButton: true,
                            message: err.message,
                          };
                        },
                        loading: "Deleting...",
                        success: {
                          closeButton: true,
                          message: "Deleted successfully!",
                        },
                      });
                    }}
                    variant="destructive"
                  >
                    Delete
                  </Button>
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      );
    },
    header: "",
    id: "actions",
  }),
];

const rateLimitConfigSchema = z.object({
  limit: z.coerce.number<number>().min(1, "Limit must be at least 1"),
  name: z.string().min(3, "Name is required"),
  window: z.string().min(2, "Window is required"),
});

function RateLimitConfigFormSheet({
  editItem,
  setEditItem,
}: {
  editItem: null | RateLimitConfig;
  setEditItem: (item: null | RateLimitConfig) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const { create, update } = useRateLimitConfigMutation();

  useEffect(() => {
    if (editItem) {
      setIsOpen(true);
    }
  }, [editItem]);

  const defaultValues = useMemo(
    () => ({
      limit: editItem?.limit ?? 1,
      name: editItem?.name ?? "1 per 5s",
      window: editItem?.window ?? "5s",
    }),
    [editItem?.limit, editItem?.name, editItem?.window],
  );

  const form = useAppForm({
    canSubmitWhenInvalid: true,
    defaultValues,
    onSubmit: async ({ value }) => {
      value = rateLimitConfigSchema.parse(value);
      if (editItem) {
        await update.mutateAsync({
          id: editItem.id,
          limit: value.limit,
          name: value.name,
          window: value.window,
        });
        toast.success("Updated successfully!");
      } else {
        await create.mutateAsync({
          limit: value.limit,
          name: value.name,
          window: value.window,
        });
        toast.success("Created successfully!");
      }
      setIsOpen(false);
    },
    validators: {
      onChange: rateLimitConfigSchema,
    },
  });

  useEffect(() => {
    form.reset(defaultValues);
  }, [defaultValues, form]);

  return (
    <Sheet onOpenChange={setIsOpen} open={isOpen}>
      <SheetTrigger asChild>
        <Button
          onClick={() => {
            setEditItem(null);
          }}
          size="sm"
        >
          <Plus className="mr-2 size-4" />
          Add Config
        </Button>
      </SheetTrigger>
      <SheetContent asChild>
        <Form form={form}>
          <SheetHeader>
            <SheetTitle>
              {editItem ? "Edit" : "Add"} Rate Limit Config
            </SheetTitle>
            <SheetDescription>
              {editItem
                ? "Update the rate limit configuration."
                : "Create a new rate limit configuration. Use duration format like 30s, 1m, 1h for the window."}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className="overflow-hidden">
            <div className="flex flex-col gap-4 px-4">
              <form.AppField name="name">
                {(field) => <field.Input label="Name" type="text" />}
              </form.AppField>
              <form.AppField name="limit">
                {(field) => <field.Input label="Limit" min={1} type="number" />}
              </form.AppField>
              <form.AppField name="window">
                {(field) => (
                  <field.Input
                    label="Window"
                    placeholder="e.g., 30s, 1m, 1h"
                    type="text"
                  />
                )}
              </form.AppField>
            </div>
          </ScrollArea>

          <SheetFooter>
            <form.AppForm>
              <form.SubmitButton className="w-full">
                {editItem ? "Update" : "Add"} Rate Limit Config
              </form.SubmitButton>
            </form.AppForm>
          </SheetFooter>
        </Form>
      </SheetContent>
    </Sheet>
  );
}

export const Route = createFileRoute("/dash/settings/ratelimit-configs")({
  component: RouteComponent,
  staticData: {
    crumb: "Rate Limit Configs",
  },
});

function RouteComponent() {
  const rateLimitConfigs = useRateLimitConfigs();
  const { remove: removeConfig } = useRateLimitConfigMutation();

  const [editItem, setEditItem] = useState<null | RateLimitConfig>(null);
  const onEditItem = (item: RateLimitConfig) => {
    setEditItem(item);
  };

  const table = useDataTable({
    columns,
    data: rateLimitConfigs.data ?? [],
    initialState: {
      columnPinning: { right: ["actions"] },
    },
    meta: {
      ctx: {
        onEdit: onEditItem,
        removeConfig,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Rate Limit Configs</h2>
        <RateLimitConfigFormSheet
          editItem={editItem}
          setEditItem={setEditItem}
        />
      </div>

      {rateLimitConfigs.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : rateLimitConfigs.isError ? (
        <div className="text-sm text-red-600">
          Error loading rate limit configs
        </div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
