import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import {
  CheckCircle,
  Pencil,
  Plus,
  Power,
  ShieldCheck,
  Trash2,
  XCircle,
} from "lucide-react";
import { DateTime } from "luxon";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import z from "zod";

import {
  UsenetServer,
  useUsenetServerMutation,
  useUsenetServers,
} from "@/api/vault-usenet-server";
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
    UsenetServer: {
      onEdit: (item: UsenetServer) => void;
      removeServer: ReturnType<typeof useUsenetServerMutation>["remove"];
      toggleServer: ReturnType<typeof useUsenetServerMutation>["toggle"];
    };
  }

  export interface DataTableMetaCtxKey {
    UsenetServer: UsenetServer;
  }
}

const col = createColumnHelper<UsenetServer>();

const columns: ColumnDef<UsenetServer>[] = [
  col.accessor("name", {
    header: "Name",
  }),
  col.accessor("host", {
    header: "Host",
  }),
  col.accessor("port", {
    header: "Port",
  }),
  col.accessor("tls", {
    cell: ({ getValue }) => {
      const tls = getValue();
      return tls ? (
        <span className="flex items-center gap-1 text-green-500">
          <CheckCircle className="size-4" />
          Yes
        </span>
      ) : (
        <span className="flex items-center gap-1 text-red-500">
          <XCircle className="size-4" />
          No
        </span>
      );
    },
    header: "TLS",
  }),
  col.accessor("priority", {
    header: "Priority",
  }),
  col.accessor("is_backup", {
    cell: ({ getValue }) => {
      const isBackup = getValue();
      return isBackup ? (
        <span className="flex items-center gap-1 text-yellow-500">
          <CheckCircle className="size-4" />
          Yes
        </span>
      ) : (
        <span className="text-muted-foreground">No</span>
      );
    },
    header: "Backup",
  }),
  col.accessor("max_connections", {
    header: "Max Conn",
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
      const { onEdit, removeServer, toggleServer } = c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                disabled={toggleServer.isPending}
                onClick={() => {
                  toast.promise(toggleServer.mutateAsync(item.id), {
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
                <AlertDialogTitle>Delete Usenet Server?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the Usenet server credentials for{" "}
                  <strong>{item.name}</strong>. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={removeServer.isPending}
                    onClick={() => {
                      toast.promise(removeServer.mutateAsync(item.id), {
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

const usenetServerSchema = z.object({
  host: z.string().min(1, "Host is required"),
  is_backup: z.boolean(),
  max_connections: z.coerce
    .number<number>()
    .int()
    .min(1, "Max connections must be at least 1"),
  name: z.string().min(1, "Name is required"),
  password: z.string(),
  port: z.coerce
    .number<number>()
    .int()
    .min(1, "Port must be at least 1")
    .max(65535, "Port must be at most 65535"),
  priority: z.string().regex(/^[0-9]$/, {
    message: "Must be between 0 to 9",
  }),
  tls: z.boolean(),
  tls_skip_verify: z.boolean(),
  username: z.string(),
});

const priorityOptions = [
  { label: "0 (Highest)", value: "0" },
  { label: "1", value: "1" },
  { label: "2", value: "2" },
  { label: "3", value: "3" },
  { label: "4", value: "4" },
  { label: "5", value: "5" },
  { label: "6", value: "6" },
  { label: "7", value: "7" },
  { label: "8", value: "8" },
  { label: "9 (Lowest)", value: "9" },
];

function UsenetServerFormSheet({
  editItem,
  open,
  setEditItem,
  setOpen,
}: {
  editItem?: null | UsenetServer;
  open: boolean;
  setEditItem: (item: null | UsenetServer) => void;
  setOpen: (open: boolean) => void;
}) {
  const { create, ping, update } = useUsenetServerMutation();
  const isEdit = Boolean(editItem);

  const form = useAppForm({
    canSubmitWhenInvalid: true,
    defaultValues: {
      host: editItem?.host ?? "",
      is_backup: editItem?.is_backup ?? false,
      max_connections: editItem?.max_connections ?? 10,
      name: editItem?.name ?? "",
      password: "",
      port: editItem?.port ?? 563,
      priority: String(editItem?.priority ?? 0),
      tls: editItem?.tls ?? true,
      tls_skip_verify: editItem?.tls_skip_verify ?? false,
      username: editItem?.username ?? "",
    },
    onSubmit: async ({ value }) => {
      value = usenetServerSchema.parse(value);
      if (editItem) {
        await update.mutateAsync({
          host: value.host,
          id: editItem.id,
          is_backup: value.is_backup,
          max_connections: value.max_connections,
          name: value.name,
          password: value.password,
          port: value.port,
          priority: Number(value.priority),
          tls: value.tls,
          tls_skip_verify: value.tls_skip_verify,
          username: value.username,
        });
        toast.success("Updated successfully!");
      } else {
        await create.mutateAsync({
          host: value.host,
          is_backup: value.is_backup,
          max_connections: value.max_connections,
          name: value.name,
          password: value.password,
          port: value.port,
          priority: Number(value.priority),
          tls: value.tls,
          tls_skip_verify: value.tls_skip_verify,
          username: value.username,
        });
        toast.success("Created successfully!");
      }
      setOpen(false);
      setEditItem(null);
    },
    validators: {
      onChange: usenetServerSchema,
    },
  });

  useEffect(() => {
    form.reset();
  }, [form, editItem]);

  const handleTestConnection = () => {
    toast.promise(
      usenetServerSchema.safeParseAsync(form.state.values).then((result) => {
        if (!result.success) {
          throw new APIError(400, {
            code: 400,
            errors: [],
            message: "Invalid Configuration",
          });
        }
        return ping.mutateAsync({
          ...result.data,
          id: editItem?.id,
        });
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Testing connection...",
        position: "bottom-left",
        success(data) {
          return {
            closeButton: true,
            message: data.message,
          };
        },
      },
    );
  };

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>
        <Button
          onClick={() => {
            setEditItem(null);
          }}
          size="sm"
        >
          <Plus className="mr-2 size-4" />
          Add Server
        </Button>
      </SheetTrigger>
      <SheetContent asChild>
        <Form form={form}>
          <SheetHeader>
            <SheetTitle>{editItem ? "Edit" : "Add"} Usenet Server</SheetTitle>
            <SheetDescription>
              {editItem ? "Update this Usenet Server." : "Add Usenet Server."}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className="overflow-hidden">
            <div className="flex flex-col gap-4 px-4">
              <form.AppField name="name">
                {(field) => <field.Input label="Name" type="text" />}
              </form.AppField>
              <form.AppField name="host">
                {(field) => <field.Input label="Host" type="text" />}
              </form.AppField>
              <form.AppField name="port">
                {(field) => <field.Input label="Port" type="number" />}
              </form.AppField>
              <form.AppField name="username">
                {(field) => <field.Input label="Username" type="text" />}
              </form.AppField>
              <form.AppField name="password">
                {(field) => (
                  <field.Input
                    label="Password"
                    placeholder={isEdit ? "<Unchanged>" : ""}
                    type="password"
                  />
                )}
              </form.AppField>
              <form.AppField name="tls">
                {(field) => <field.Checkbox label="Use TLS" />}
              </form.AppField>
              <form.AppField name="tls_skip_verify">
                {(field) => (
                  <form.Subscribe selector={(o) => o.values.tls}>
                    {(tls) => (
                      <field.Checkbox
                        disabled={!tls}
                        label="Skip TLS Verification"
                      />
                    )}
                  </form.Subscribe>
                )}
              </form.AppField>
              <form.AppField name="priority">
                {(field) => (
                  <field.Select
                    label="Priority"
                    options={priorityOptions}
                    placeholder="Select priority"
                  />
                )}
              </form.AppField>
              <form.AppField name="is_backup">
                {(field) => <field.Checkbox label="Backup" />}
              </form.AppField>
              <form.AppField name="max_connections">
                {(field) => (
                  <field.Input label="Max Connections" type="number" />
                )}
              </form.AppField>
            </div>
          </ScrollArea>

          <SheetFooter>
            <Button
              className="w-full"
              disabled={ping.isPending}
              onClick={handleTestConnection}
              type="button"
              variant="outline"
            >
              <ShieldCheck className="mr-2 size-4" />
              Test Connection
            </Button>
            <form.SubmitButton className="w-full">Save</form.SubmitButton>
          </SheetFooter>
        </Form>
      </SheetContent>
    </Sheet>
  );
}

export const Route = createFileRoute("/dash/usenet/servers")({
  component: RouteComponent,
  staticData: {
    crumb: "Servers",
  },
});

function RouteComponent() {
  const usenetServers = useUsenetServers();
  const { remove: removeServer, toggle: toggleServer } =
    useUsenetServerMutation();

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editItem, setEditItem] = useState<null | UsenetServer>(null);

  const onEditItem = (item: UsenetServer) => {
    setEditItem(item);
    setSheetOpen(true);
  };

  const table = useDataTable({
    columns,
    data: usenetServers.data ?? [],
    initialState: {
      columnPinning: { left: ["name"], right: ["actions"] },
    },
    meta: {
      ctx: {
        onEdit: onEditItem,
        removeServer,
        toggleServer,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Usenet Servers</h2>
        <UsenetServerFormSheet
          editItem={editItem}
          open={sheetOpen}
          setEditItem={setEditItem}
          setOpen={setSheetOpen}
        />
      </div>

      {usenetServers.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : usenetServers.isError ? (
        <div className="text-sm text-red-600">Error loading Usenet Servers</div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
