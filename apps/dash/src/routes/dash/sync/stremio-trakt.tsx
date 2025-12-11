import { createFileRoute, Link } from "@tanstack/react-router";
import {
  ArrowLeftRight,
  ArrowRight,
  CheckCircle,
  Link2,
  Plus,
  RefreshCw,
  Trash2,
  XCircle,
} from "lucide-react";
import { DateTime } from "luxon";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import {
  StremioTraktLink,
  SyncDirection,
  useStremioTraktLinkMutation,
  useStremioTraktLinks,
} from "@/api/sync-stremio-trakt";
import {
  StremioAccount,
  useStremioAccounts,
} from "@/api/vault-stremio-account";
import { TraktAccount, useTraktAccounts } from "@/api/vault-trakt-account";
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
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { APIError } from "@/lib/api";

export const Route = createFileRoute("/dash/sync/stremio-trakt")({
  component: RouteComponent,
  staticData: {
    crumb: "Stremio ↔ Trakt",
  },
});

const syncDirectionOptions: Array<{
  icon: typeof ArrowRight;
  label: string;
  value: SyncDirection;
}> = [
  {
    icon: XCircle,
    label: "Disabled",
    value: "none",
  },
  {
    icon: ArrowRight,
    label: "Stremio → Trakt",
    value: "stremio_to_trakt",
  },
  {
    icon: ArrowRight,
    label: "Trakt → Stremio",
    value: "trakt_to_stremio",
  },
  {
    icon: ArrowLeftRight,
    label: "Bidirectional",
    value: "both",
  },
];

function LinkAccountSheet({
  onClose,
  stremioAccounts,
  traktAccounts,
}: {
  onClose: () => void;
  stremioAccounts: StremioAccount[];
  traktAccounts: TraktAccount[];
}) {
  const { create } = useStremioTraktLinkMutation();

  const availableStremioAccounts = stremioAccounts;
  const availableTraktAccounts = traktAccounts;

  const form = useAppForm({
    defaultValues: {
      stremio_account_id: "",
      trakt_account_id: "",
    },
    onSubmit: async ({ value }) => {
      await create.mutateAsync({
        stremio_account_id: value.stremio_account_id,
        sync_config: { watched: { dir: "none" } },
        trakt_account_id: value.trakt_account_id,
      });
      toast.success("Accounts linked successfully!");
      onClose();
    },
  });

  return (
    <Form className="flex flex-col gap-4" form={form}>
      <form.AppField name="stremio_account_id">
        {(field) => (
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium" htmlFor={field.name}>
              Stremio Account
            </label>
            {availableStremioAccounts.length === 0 ? (
              <div className="text-muted-foreground text-sm">
                No available Stremio accounts.{" "}
                <Link
                  className="text-primary underline underline-offset-4"
                  to="/dash/vault/stremio-accounts"
                >
                  Add one in Vault
                </Link>
                .
              </div>
            ) : (
              <Select
                onValueChange={(value) => field.handleChange(value)}
                value={field.state.value}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select Stremio account" />
                </SelectTrigger>
                <SelectContent>
                  {availableStremioAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id}>
                      {account.email}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        )}
      </form.AppField>

      <form.AppField name="trakt_account_id">
        {(field) => (
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium" htmlFor={field.name}>
              Trakt Account
            </label>
            {availableTraktAccounts.length === 0 ? (
              <div className="text-muted-foreground text-sm">
                No available Trakt accounts.{" "}
                <Link
                  className="text-primary underline underline-offset-4"
                  to="/dash/vault/trakt-accounts"
                >
                  Add one in Vault
                </Link>
                .
              </div>
            ) : (
              <Select
                onValueChange={(value) => field.handleChange(value)}
                value={field.state.value}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select Trakt account" />
                </SelectTrigger>
                <SelectContent>
                  {availableTraktAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id}>
                      {account.user_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        )}
      </form.AppField>

      <form.AppForm>
        <form.SubmitButton
          className="w-full"
          disabled={
            availableStremioAccounts.length === 0 ||
            availableTraktAccounts.length === 0
          }
        >
          Link Accounts
        </form.SubmitButton>
      </form.AppForm>
    </Form>
  );
}

function LinkCard({
  link,
  stremioAccount,
  traktAccount,
}: {
  link: StremioTraktLink;
  stremioAccount?: StremioAccount;
  traktAccount?: TraktAccount;
}) {
  const { remove, resetSyncState, sync, update } =
    useStremioTraktLinkMutation();

  const selectedWatchedSyncDirection = syncDirectionOptions.find(
    (opt) => opt.value === link.sync_config.watched.dir,
  );
  const SyncDirectionIcon = selectedWatchedSyncDirection?.icon || XCircle;

  const handleWatchedSyncDirectionChange = (value: string) => {
    toast.promise(
      update.mutateAsync({
        stremio_account_id: link.stremio_account_id,
        sync_config: { watched: { dir: value as SyncDirection } },
        trakt_account_id: link.trakt_account_id,
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Updating sync direction...",
        success: {
          closeButton: true,
          message: "Sync direction updated!",
        },
      },
    );
  };

  const handleSync = () => {
    toast.promise(
      sync.mutateAsync({
        stremio_account_id: link.stremio_account_id,
        trakt_account_id: link.trakt_account_id,
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Triggering sync...",
        success: {
          closeButton: true,
          message: "Sync triggered!",
        },
      },
    );
  };

  const handleUnlink = () => {
    toast.promise(
      remove.mutateAsync({
        stremio_account_id: link.stremio_account_id,
        trakt_account_id: link.trakt_account_id,
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Unlinking...",
        success: {
          closeButton: true,
          message: "Accounts unlinked!",
        },
      },
    );
  };

  const handleResetSyncState = () => {
    toast.promise(
      resetSyncState.mutateAsync({
        stremio_account_id: link.stremio_account_id,
        trakt_account_id: link.trakt_account_id,
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Resetting sync status...",
        success: {
          closeButton: true,
          message: "Sync status reset! Next sync will be a full sync.",
        },
      },
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Link2 className="size-4" />
          Linked Accounts
        </CardTitle>
        <CardDescription>
          <div className="flex flex-col gap-1">
            <div>
              <span className="font-medium">Stremio:</span>{" "}
              {stremioAccount?.email || link.stremio_account_id}
            </div>
            <div>
              <span className="font-medium">Trakt:</span>{" "}
              {traktAccount?.user_name || link.trakt_account_id}
            </div>
          </div>
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium">Watched Sync Direction</label>
          <Select
            onValueChange={handleWatchedSyncDirectionChange}
            value={link.sync_config.watched.dir}
          >
            <SelectTrigger className="w-full">
              <SelectValue>
                <div className="flex items-center gap-2">
                  <SyncDirectionIcon className="size-4" />
                  {selectedWatchedSyncDirection?.label}
                </div>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {syncDirectionOptions.map((option) => {
                const OptionIcon = option.icon;
                return (
                  <SelectItem key={option.value} value={option.value}>
                    <div className="flex items-center gap-2">
                      <OptionIcon className="size-4" />
                      {option.label}
                    </div>
                  </SelectItem>
                );
              })}
            </SelectContent>
          </Select>
        </div>

        {link.sync_state.watched.last_synced_at && (
          <div className="text-muted-foreground flex flex-col gap-1 text-sm">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1">
                <CheckCircle className="size-3.5 text-green-500" />
                <span>
                  Last synced:{" "}
                  {DateTime.fromISO(
                    link.sync_state.watched.last_synced_at,
                  ).toLocaleString(DateTime.DATETIME_MED)}
                </span>
              </div>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button size="sm" variant="ghost">
                    Reset
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Reset Sync Status?</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will clear the last sync timestamp and force a full
                      re-sync on the next sync operation. This can be useful if
                      you suspect the sync is incomplete or has missing items.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction asChild>
                      <Button
                        disabled={resetSyncState.isPending}
                        onClick={handleResetSyncState}
                      >
                        Reset
                      </Button>
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        )}
      </CardContent>
      <CardFooter className="mt-auto gap-4">
        <Button
          className="flex-1"
          disabled={link.sync_config.watched.dir === "none" || sync.isPending}
          onClick={handleSync}
          size="sm"
          variant="outline"
        >
          <RefreshCw className="mr-2 size-4" />
          Sync Now
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button size="sm" variant="outline">
              <Trash2 className="text-destructive mr-2 size-4" />
              Unlink
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Unlink Accounts?</AlertDialogTitle>
              <AlertDialogDescription>
                This will remove the link between{" "}
                <strong>
                  {stremioAccount?.email || "this Stremio account"}
                </strong>{" "}
                and{" "}
                <strong>
                  {traktAccount?.user_name || "this Trakt account"}
                </strong>
                . Sync will stop, but your watch history won't be deleted.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction asChild>
                <Button
                  disabled={remove.isPending}
                  onClick={handleUnlink}
                  variant="destructive"
                >
                  Unlink
                </Button>
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardFooter>
    </Card>
  );
}

function RouteComponent() {
  const links = useStremioTraktLinks();
  const stremioAccounts = useStremioAccounts();
  const traktAccounts = useTraktAccounts();

  const [sheetOpen, setSheetOpen] = useState(false);

  const stremioAccountsById = useMemo(
    () => new Map(stremioAccounts.data?.map((acc) => [acc.id, acc])),
    [stremioAccounts.data],
  );
  const traktAccountsById = useMemo(
    () => new Map(traktAccounts.data?.map((acc) => [acc.id, acc])),
    [traktAccounts.data],
  );

  const isLoading =
    links.isLoading || stremioAccounts.isLoading || traktAccounts.isLoading;
  const hasError =
    links.isError || stremioAccounts.isError || traktAccounts.isError;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Stremio ↔ Trakt Sync</h2>
          <p className="text-muted-foreground text-sm">
            Link Stremio and Trakt accounts to sync watch history
          </p>
        </div>
        <Sheet onOpenChange={setSheetOpen} open={sheetOpen}>
          <SheetTrigger asChild>
            <Button size="sm">
              <Plus className="mr-2 size-4" />
              Link Accounts
            </Button>
          </SheetTrigger>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>Link Accounts</SheetTitle>
              <SheetDescription>
                Choose which Stremio and Trakt accounts to link for sync.
              </SheetDescription>
            </SheetHeader>
            <div className="p-4">
              {stremioAccounts.data && traktAccounts.data && links.data ? (
                <LinkAccountSheet
                  onClose={() => setSheetOpen(false)}
                  stremioAccounts={stremioAccounts.data}
                  traktAccounts={traktAccounts.data}
                />
              ) : (
                <div className="text-muted-foreground text-sm">Loading...</div>
              )}
            </div>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : hasError ? (
        <div className="text-sm text-red-600">Error loading data</div>
      ) : links.data?.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-4 py-12">
            <Link2 className="text-muted-foreground size-12" />
            <div className="flex flex-col items-center gap-2 text-center">
              <h3 className="font-semibold">No linked accounts</h3>
              <p className="text-muted-foreground text-sm">
                Link your Stremio and Trakt accounts to start syncing watch
                history
              </p>
            </div>
            {(stremioAccounts.data?.length === 0 ||
              traktAccounts.data?.length === 0) && (
              <div className="text-muted-foreground flex flex-col gap-1 text-sm">
                {stremioAccounts.data?.length === 0 && (
                  <div>
                    Add a{" "}
                    <Link
                      className="text-primary underline underline-offset-4"
                      to="/dash/vault/stremio-accounts"
                    >
                      Stremio account
                    </Link>
                  </div>
                )}
                {traktAccounts.data?.length === 0 && (
                  <div>
                    Add a{" "}
                    <Link
                      className="text-primary underline underline-offset-4"
                      to="/dash/vault/trakt-accounts"
                    >
                      Trakt account
                    </Link>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {links.data?.map((link) => (
            <LinkCard
              key={`${link.stremio_account_id}:${link.trakt_account_id}`}
              link={link}
              stremioAccount={stremioAccountsById.get(link.stremio_account_id)}
              traktAccount={traktAccountsById.get(link.trakt_account_id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
