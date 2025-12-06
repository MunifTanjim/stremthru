import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import {
  CheckCircle,
  Pencil,
  Plus,
  RefreshCwIcon,
  Trash2,
  XCircle,
} from "lucide-react";
import { DateTime } from "luxon";
import { useState } from "react";
import { toast } from "sonner";

import {
  StremioAccount,
  useStremioAccountMutation,
  useStremioAccounts,
} from "@/api/vault-stremio-account";
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
    StremioAccount: {
      getAccount: ReturnType<typeof useStremioAccountMutation>["get"];
      onEdit: (item: StremioAccount) => void;
      removeAccount: ReturnType<typeof useStremioAccountMutation>["remove"];
    };
  }

  export interface DataTableMetaCtxKey {
    StremioAccount: StremioAccount;
  }
}

const col = createColumnHelper<StremioAccount>();

const columns: ColumnDef<StremioAccount>[] = [
  col.accessor("email", {
    header: "Email",
  }),
  col.accessor("is_valid", {
    cell: ({ getValue }) => {
      const isValid = getValue();
      return isValid ? (
        <span className="flex items-center gap-1 text-green-500">
          <CheckCircle className="size-4" />
          Valid
        </span>
      ) : (
        <span className="flex items-center gap-1 text-red-500">
          <XCircle className="size-4" />
          Invalid
        </span>
      );
    },
    header: "Validity",
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
      const { getAccount, onEdit, removeAccount } = c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                disabled={getAccount.isPending}
                onClick={() => {
                  toast.promise(
                    getAccount.mutateAsync({ id: item.id, refresh: true }),
                    {
                      error(err: APIError) {
                        console.error(err);
                        return {
                          closeButton: true,
                          message: err.message,
                        };
                      },
                      loading: "Refreshing account...",
                      success: {
                        closeButton: true,
                        message: "Refreshed account!",
                      },
                    },
                  );
                }}
                size="icon-sm"
                variant="ghost"
              >
                <RefreshCwIcon />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Refresh</TooltipContent>
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
                <AlertDialogTitle>Delete Stremio Account?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the Stremio account credentials
                  for <strong>{item.email}</strong>. This action cannot be
                  undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={removeAccount.isPending}
                    onClick={() => {
                      toast.promise(removeAccount.mutateAsync(item.id), {
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

function StremioAccountForm({
  editItem,
  onClose,
}: {
  editItem?: null | StremioAccount;
  onClose: () => void;
}) {
  const { create, update } = useStremioAccountMutation();
  const isEdit = Boolean(editItem);

  const form = useAppForm({
    canSubmitWhenInvalid: true,
    defaultValues: {
      email: editItem?.email ?? "",
      password: "",
    },
    onSubmit: async ({ value }) => {
      if (isEdit && editItem) {
        await update.mutateAsync({
          id: editItem.id,
          password: value.password,
        });
        toast.success("Updated successfully!");
      } else {
        await create.mutateAsync({
          email: value.email,
          password: value.password,
        });
        toast.success("Created successfully!");
      }
      onClose();
    },
  });

  return (
    <Form className="flex flex-col gap-4" form={form}>
      <form.AppField name="email">
        {(field) => (
          <field.Input disabled={isEdit} label="Email" type="email" />
        )}
      </form.AppField>
      <form.AppField name="password">
        {(field) => <field.Input label="Password" type="password" />}
      </form.AppField>
      <form.AppForm>
        <form.SubmitButton className="w-full">
          {isEdit ? "Update" : "Add"} Stremio Account
        </form.SubmitButton>
      </form.AppForm>
    </Form>
  );
}

export const Route = createFileRoute("/dash/vault/stremio-accounts")({
  component: RouteComponent,
  staticData: {
    crumb: "Stremio Accounts",
  },
});

function RouteComponent() {
  const stremioAccounts = useStremioAccounts();
  const { get: getAccount, remove: removeAccount } =
    useStremioAccountMutation();

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editItem, setEditItem] = useState<null | StremioAccount>(null);

  const handleEdit = (item: StremioAccount) => {
    setEditItem(item);
    setSheetOpen(true);
  };

  const handleClose = () => {
    setSheetOpen(false);
    setEditItem(null);
  };

  const table = useDataTable({
    columns,
    data: stremioAccounts.data ?? [],
    initialState: {
      columnPinning: { right: ["actions"] },
    },
    meta: {
      ctx: {
        getAccount,
        onEdit: handleEdit,
        removeAccount,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Stremio Accounts</h2>
        <Sheet onOpenChange={setSheetOpen} open={sheetOpen}>
          <SheetTrigger asChild>
            <Button
              onClick={() => {
                setEditItem(null);
              }}
              size="sm"
            >
              <Plus className="mr-2 size-4" />
              Add Account
            </Button>
          </SheetTrigger>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>
                {editItem ? "Edit" : "Add"} Stremio Account
              </SheetTitle>
              <SheetDescription>
                {editItem
                  ? "Update the credentials for this Stremio account."
                  : "Add your Stremio account credentials. The password will be encrypted before storage."}
              </SheetDescription>
            </SheetHeader>
            <div className="p-4">
              <StremioAccountForm editItem={editItem} onClose={handleClose} />
            </div>
          </SheetContent>
        </Sheet>
      </div>

      {stremioAccounts.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : stremioAccounts.isError ? (
        <div className="text-sm text-red-600">
          Error loading Stremio accounts
        </div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
