import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import {
  CheckCircle,
  Pencil,
  Plus,
  Power,
  RefreshCwIcon,
  Trash2,
  XCircle,
} from "lucide-react";
import { DateTime } from "luxon";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  useRateLimitConfig,
  useRateLimitConfigs,
} from "@/api/ratelimit-config";
import {
  NewznabIndexer,
  useNewznabIndexerMutation,
  useNewznabIndexers,
} from "@/api/vault-newznab-indexer";
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
    NewznabIndexer: {
      onEdit: (item: NewznabIndexer) => void;
      removeIndexer: ReturnType<typeof useNewznabIndexerMutation>["remove"];
      testIndexer: ReturnType<typeof useNewznabIndexerMutation>["test"];
      toggleIndexer: ReturnType<typeof useNewznabIndexerMutation>["toggle"];
    };
  }

  export interface DataTableMetaCtxKey {
    NewznabIndexer: NewznabIndexer;
  }
}

const col = createColumnHelper<NewznabIndexer>();

function RateLimitConfigName({ id }: { id: null | string }) {
  const conf = useRateLimitConfig(id);
  return conf ? conf.name : "-";
}

const columns: ColumnDef<NewznabIndexer>[] = [
  col.accessor("name", {
    header: "Name",
  }),
  col.accessor("url", {
    cell: ({ getValue }) => {
      const url = getValue();
      return <span className="max-w-md truncate font-mono text-xs">{url}</span>;
    },
    header: "URL",
  }),
  col.accessor("rate_limit_config_id", {
    cell: ({ getValue }) => {
      return <RateLimitConfigName id={getValue()} />;
    },
    header: "Rate Limit",
  }),
  col.accessor("disabled", {
    cell: ({ getValue }) => {
      const disabled = getValue();
      return disabled ? (
        <span className="flex items-center gap-1 text-red-500">
          <XCircle className="size-4" />
          Disabled
        </span>
      ) : (
        <span className="flex items-center gap-1 text-green-500">
          <CheckCircle className="size-4" />
          Enabled
        </span>
      );
    },
    header: "Status",
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
      const { onEdit, removeIndexer, testIndexer, toggleIndexer } =
        c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                disabled={toggleIndexer.isPending}
                onClick={() => {
                  toast.promise(toggleIndexer.mutateAsync(item.id), {
                    error(err: APIError) {
                      console.error(err);
                      return {
                        closeButton: true,
                        message: err.message,
                      };
                    },
                    loading: item.disabled ? "Enabling..." : "Disabling...",
                    success: {
                      closeButton: true,
                      message: item.disabled
                        ? "Enabled successfully!"
                        : "Disabled successfully!",
                    },
                  });
                }}
                size="icon-sm"
                variant="ghost"
              >
                <Power
                  className={item.disabled ? "text-red-500" : "text-green-500"}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {item.disabled ? "Enable" : "Disable"}
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                disabled={testIndexer.isPending}
                onClick={() => {
                  toast.promise(testIndexer.mutateAsync(item.id), {
                    error(err: APIError) {
                      console.error(err);
                      return {
                        closeButton: true,
                        message: err.message,
                      };
                    },
                    loading: "Testing connection...",
                    success: {
                      closeButton: true,
                      message: "Connection test successful!",
                    },
                  });
                }}
                size="icon-sm"
                variant="ghost"
              >
                <RefreshCwIcon />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Test Connection</TooltipContent>
          </Tooltip>
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
                <AlertDialogTitle>Delete Newznab Indexer?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the Newznab indexer{" "}
                  <strong>{item.name}</strong>. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={removeIndexer.isPending}
                    onClick={() => {
                      toast.promise(
                        removeIndexer.mutateAsync({ id: item.id }),
                        {
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
                        },
                      );
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

function NewznabIndexerFormSheet({
  editItem,
  setEditItem,
}: {
  editItem: NewznabIndexer | null;
  setEditItem: (item: NewznabIndexer | null) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const rateLimitConfigs = useRateLimitConfigs();
  const rateLimitConfigOptions = useMemo(() => {
    return (rateLimitConfigs.data ?? [])?.map((config) => ({
      label: config.name,
      value: config.id,
    }));
  }, [rateLimitConfigs.data]);

  useEffect(() => {
    if (editItem) {
      setIsOpen(true);
    }
  }, [editItem]);

  const { create, update } = useNewznabIndexerMutation();

  const defaultValues = useMemo(
    () => ({
      api_key: "",
      name: editItem?.name ?? "",
      rate_limit_config_id: editItem?.rate_limit_config_id ?? "",
      url: editItem?.url ?? "",
    }),
    [editItem?.name, editItem?.rate_limit_config_id, editItem?.url],
  );

  const form = useAppForm({
    defaultValues,
    onSubmit: async ({ value }) => {
      if (editItem) {
        await update.mutateAsync({
          api_key: value.api_key,
          id: editItem.id,
          name: value.name,
          rate_limit_config_id: value.rate_limit_config_id || null,
        });
        toast.success("Updated successfully!");
      } else {
        await create.mutateAsync({
          api_key: value.api_key,
          name: value.name,
          rate_limit_config_id: value.rate_limit_config_id || null,
          url: value.url,
        });
        toast.success("Created successfully!");
      }
      setIsOpen(false);
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
          Add Indexer
        </Button>
      </SheetTrigger>
      <SheetContent asChild>
        <Form form={form}>
          <SheetHeader>
            <SheetTitle>{editItem ? "Edit" : "Add"} Newznab Indexer</SheetTitle>
            <SheetDescription>
              {editItem
                ? "Update the API key for this Newznab indexer."
                : "Add a Newznab indexer (e.g., NZBGeek, DrunkenSlug). The API key will be encrypted before storage."}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className="overflow-hidden">
            <div className="flex flex-col gap-4 px-4">
              <form.AppField name="name">
                {(field) => <field.Input label="Name" type="text" />}
              </form.AppField>
              <form.AppField name="url">
                {(field) => (
                  <field.Input
                    disabled={Boolean(editItem)}
                    label="Newznab URL"
                  />
                )}
              </form.AppField>
              <form.AppField name="api_key">
                {(field) => <field.Input label="API Key" type="password" />}
              </form.AppField>
              <form.AppField name="rate_limit_config_id">
                {(field) => (
                  <field.Select
                    label="Rate Limit Config"
                    options={rateLimitConfigOptions}
                  />
                )}
              </form.AppField>
            </div>
          </ScrollArea>

          <SheetFooter>
            <form.SubmitButton className="w-full">
              {editItem ? "Update" : "Add"} Newznab Indexer
            </form.SubmitButton>
          </SheetFooter>
        </Form>
      </SheetContent>
    </Sheet>
  );
}

export const Route = createFileRoute("/dash/vault/newznab-indexers")({
  component: RouteComponent,
  staticData: {
    crumb: "Newznab Indexers",
  },
});

function RouteComponent() {
  const newznabIndexers = useNewznabIndexers();
  const {
    remove: removeIndexer,
    test: testIndexer,
    toggle: toggleIndexer,
  } = useNewznabIndexerMutation();

  const [editItem, setEditItem] = useState<NewznabIndexer | null>(null);

  const handleEdit = (item: NewznabIndexer) => {
    setEditItem(item);
  };

  const table = useDataTable({
    columns,
    data: newznabIndexers.data ?? [],
    initialState: {
      columnPinning: { left: ["name"], right: ["actions"] },
    },
    meta: {
      ctx: {
        onEdit: handleEdit,
        removeIndexer,
        testIndexer,
        toggleIndexer,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Newznab Indexers</h2>
        <NewznabIndexerFormSheet
          editItem={editItem}
          setEditItem={setEditItem}
        />
      </div>

      {newznabIndexers.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : newznabIndexers.isError ? (
        <div className="text-sm text-red-600">
          Error loading Newznab indexers
        </div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
