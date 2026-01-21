import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import {
  CheckCircle,
  Plus,
  RefreshCwIcon,
  Trash2,
  XCircle,
} from "lucide-react";
import { DateTime } from "luxon";
import { useCallback, useEffect, useRef, useState } from "react";
import { useInterval } from "react-use";
import { toast } from "sonner";

import {
  getTraktAuthURL,
  TraktAccount,
  useTraktAccountMutation,
  useTraktAccounts,
} from "@/api/vault-trakt-account";
import { DataTable } from "@/components/data-table";
import { useDataTable } from "@/components/data-table/use-data-table";
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
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    TraktAccount: {
      getAccount: ReturnType<typeof useTraktAccountMutation>["get"];
      removeAccount: ReturnType<typeof useTraktAccountMutation>["remove"];
    };
  }

  export interface DataTableMetaCtxKey {
    TraktAccount: TraktAccount;
  }
}

const col = createColumnHelper<TraktAccount>();

const columns: ColumnDef<TraktAccount>[] = [
  col.accessor("id", {
    header: "User ID",
  }),
  col.accessor("user_name", {
    header: "Username",
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
      const { getAccount, removeAccount } = c.table.options.meta!.ctx;
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
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button size="icon-sm" variant="ghost">
                <Trash2 className="text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Trakt Account?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will remove the Trakt account{" "}
                  <strong>{item.user_name}</strong> from the vault. This action
                  cannot be undone.
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

export const Route = createFileRoute("/dash/vault/trakt-accounts")({
  component: RouteComponent,
  staticData: {
    crumb: "Trakt Accounts",
  },
});

function RouteComponent() {
  const traktAccounts = useTraktAccounts();
  const {
    create: createAccount,
    get: getAccount,
    remove: removeAccount,
  } = useTraktAccountMutation();

  const [oauthState, setOauthState] = useState("");
  const popupRef = useRef<null | Window>(null);

  const handleAddAccount = useCallback(async () => {
    try {
      const oauthState = `trakt-${Math.random()}`;
      setOauthState(oauthState);
      const authURL = await getTraktAuthURL(oauthState);

      const width = 600;
      const height = 700;
      const left = window.screenX + (window.outerWidth - width) / 2;
      const top = window.screenY + (window.outerHeight - height) / 2;

      popupRef.current = window.open(
        authURL,
        "vault_trakt_account_oauth",
        `width=${width},height=${height},left=${left},top=${top},popup=yes`,
      );
    } catch (err) {
      toast.error("Failed to get Trakt auth URL");
      console.error(err);
    }
  }, []);

  useInterval(
    () => {
      if (!popupRef.current || popupRef.current.closed) {
        setOauthState("");
        popupRef.current = null;
      }
    },
    oauthState ? 1000 : null,
  );

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (
        event.data?.type === "oauth_callback" &&
        event.data?.state === oauthState
      ) {
        const code = event.data.code;
        if (code) {
          toast.promise(createAccount.mutateAsync({ oauth_token_id: code }), {
            error(err: APIError) {
              console.error(err);
              return {
                closeButton: true,
                message: err.message,
              };
            },
            loading: "Adding account...",
            success: {
              closeButton: true,
              message: "Account added successfully!",
            },
          });
        }

        if (popupRef.current) {
          popupRef.current.close();
          popupRef.current = null;
          setOauthState("");
        }
      }
    };

    window.addEventListener("message", handleMessage);

    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [createAccount, oauthState]);

  const table = useDataTable({
    columns,
    data: traktAccounts.data ?? [],
    initialState: {
      columnPinning: { right: ["actions"] },
    },
    meta: {
      ctx: {
        getAccount,
        removeAccount,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Trakt Accounts</h2>
        <Button
          disabled={Boolean(oauthState)}
          onClick={handleAddAccount}
          size="sm"
        >
          {oauthState ? <Spinner /> : <Plus className="mr-2 size-4" />}
          Add Account
        </Button>
      </div>

      {traktAccounts.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : traktAccounts.isError ? (
        <div className="text-sm text-red-600">Error loading Trakt accounts</div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
