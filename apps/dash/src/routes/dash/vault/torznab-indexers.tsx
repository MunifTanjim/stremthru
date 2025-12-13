import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { Pencil, Plus, RefreshCwIcon, Trash2 } from "lucide-react";
import { DateTime } from "luxon";
import { useState } from "react";
import { toast } from "sonner";

import {
  TorznabIndexer,
  useTorznabIndexerMutation,
  useTorznabIndexers,
} from "@/api/vault-torznab-indexer";
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
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
    TorznabIndexer: {
      onEdit: (item: TorznabIndexer) => void;
      removeIndexer: ReturnType<typeof useTorznabIndexerMutation>["remove"];
      testIndexer: ReturnType<typeof useTorznabIndexerMutation>["test"];
    };
  }

  export interface DataTableMetaCtxKey {
    TorznabIndexer: TorznabIndexer;
  }
}

const col = createColumnHelper<TorznabIndexer>();

const columns: ColumnDef<TorznabIndexer>[] = [
  col.accessor("type", {
    header: "Type",
  }),
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
  col.accessor("updated_at", {
    cell: ({ getValue }) => {
      const date = DateTime.fromISO(getValue());
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Updated At",
  }),
  col.display({
    cell: (c) => {
      const { onEdit, removeIndexer, testIndexer } = c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
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
                <AlertDialogTitle>Delete Torznab Indexer?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the Torznab indexer{" "}
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

function TorznabIndexerForm({
  editItem,
  onClose,
}: {
  editItem?: null | TorznabIndexer;
  onClose: () => void;
}) {
  const { create, update } = useTorznabIndexerMutation();

  const form = useAppForm({
    defaultValues: {
      api_key: "",
      name: editItem?.name ?? "",
      url: editItem?.url ?? "",
    },
    onSubmit: async ({ value }) => {
      if (editItem) {
        await update.mutateAsync({
          api_key: value.api_key,
          id: editItem.id,
          name: value.name,
        });
        toast.success("Updated successfully!");
      } else {
        await create.mutateAsync({
          api_key: value.api_key,
          name: value.name,
          url: value.url,
        });
        toast.success("Created successfully!");
      }
      onClose();
    },
  });

  return (
    <Form className="flex flex-col gap-4" form={form}>
      <form.AppField name="name">
        {(field) => <field.Input label="Name" type="text" />}
      </form.AppField>
      <form.AppField name="url">
        {(field) => (
          <field.Input disabled={Boolean(editItem)} label="Torznab URL" />
        )}
      </form.AppField>
      <form.AppField name="api_key">
        {(field) => <field.Input label="API Key" type="password" />}
      </form.AppField>
      <form.AppForm>
        <form.SubmitButton className="w-full">
          {editItem ? "Update" : "Add"} Torznab Indexer
        </form.SubmitButton>
      </form.AppForm>
    </Form>
  );
}

export const Route = createFileRoute("/dash/vault/torznab-indexers")({
  component: RouteComponent,
  staticData: {
    crumb: "Torznab Indexers",
  },
});

function RouteComponent() {
  const torznabIndexers = useTorznabIndexers();
  const { remove: removeIndexer, test: testIndexer } =
    useTorznabIndexerMutation();

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editItem, setEditItem] = useState<null | TorznabIndexer>(null);

  const handleEdit = (item: TorznabIndexer) => {
    setEditItem(item);
    setSheetOpen(true);
  };

  const handleClose = () => {
    setSheetOpen(false);
    setEditItem(null);
  };

  const table = useDataTable({
    columns,
    data: torznabIndexers.data ?? [],
    initialState: {
      columnPinning: { left: ["name"], right: ["actions"] },
    },
    meta: {
      ctx: {
        onEdit: handleEdit,
        removeIndexer,
        testIndexer,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Torznab Indexers</h2>
        <Sheet onOpenChange={setSheetOpen} open={sheetOpen}>
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
          <SheetContent>
            <SheetHeader>
              <SheetTitle>
                {editItem ? "Edit" : "Add"} Torznab Indexer
              </SheetTitle>
              <SheetDescription>
                {editItem
                  ? "Update the API key for this Torznab indexer."
                  : "Add a Jackett indexer. The API key will be encrypted before storage."}
              </SheetDescription>
            </SheetHeader>
            <div className="p-4">
              <TorznabIndexerForm editItem={editItem} onClose={handleClose} />
            </div>
          </SheetContent>
        </Sheet>
      </div>

      {torznabIndexers.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : torznabIndexers.isError ? (
        <div className="text-sm text-red-600">
          Error loading Torznab indexers
        </div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
