import { createFileRoute, Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  ArrowLeftRight,
  ArrowRight,
  CheckCircle,
  ExternalLinkIcon,
  Link2,
  Plus,
  RefreshCw,
  Trash2,
  XCircle,
  XIcon,
} from "lucide-react";
import { DateTime } from "luxon";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { IMDBTitle } from "@/api/imdb";
import {
  StremioStremioLink,
  SyncDirection,
  useStremioStremioLinkMutation,
  useStremioStremioLinks,
} from "@/api/sync-stremio-stremio";
import {
  StremioAccount,
  useStremioAccounts,
} from "@/api/vault-stremio-account";
import { Form } from "@/components/form/Form";
import { useAppForm } from "@/components/form/hook";
import { IMDBSearch } from "@/components/imdb-search";
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
  Item,
  ItemActions,
  ItemContent,
  ItemGroup,
  ItemTitle,
} from "@/components/ui/item";
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

export const Route = createFileRoute("/dash/sync/stremio-stremio")({
  component: RouteComponent,
  staticData: {
    crumb: "Stremio ↔ Stremio",
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
    label: "Account A → Account B",
    value: "a_to_b",
  },
  {
    icon: ArrowLeft,
    label: "Account A ← Account B",
    value: "b_to_a",
  },
  {
    icon: ArrowLeftRight,
    label: "Bidirectional",
    value: "both",
  },
];

function ItemsManager({
  ids,
  onIdsChange,
}: {
  ids: string[];
  onIdsChange: (items: string[]) => void;
}) {
  const itemsSet = useMemo(() => new Set(ids), [ids]);

  const handleAddItem = (item: IMDBTitle) => {
    if (!itemsSet.has(item.id)) {
      onIdsChange([...ids, item.id]);
    }
  };

  const handleRemoveItem = (id: string) => {
    onIdsChange(ids.filter((tid) => tid !== id));
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <IMDBSearch onSelect={handleAddItem} />
      </div>

      {ids.length > 0 && (
        <div className="flex flex-col gap-2">
          <div className="text-muted-foreground text-sm">
            {ids.length} item{ids.length !== 1 ? "s" : ""} selected
          </div>
          <ItemGroup className="flex-row flex-wrap gap-2">
            {ids.map((id) => (
              <Item
                className="w-auto flex-shrink"
                key={id}
                size="sm"
                variant="outline"
              >
                <ItemContent>
                  <ItemTitle>{id}</ItemTitle>
                </ItemContent>
                <ItemActions>
                  <Button asChild size="icon-sm" variant="ghost">
                    <a href={`http://imdb.com/title/${id}`} target="_blank">
                      <ExternalLinkIcon />
                    </a>
                  </Button>
                  <Button
                    onClick={() => handleRemoveItem(id)}
                    size="icon-sm"
                    variant="ghost"
                  >
                    <XIcon />
                  </Button>
                </ItemActions>
              </Item>
            ))}
          </ItemGroup>
        </div>
      )}
    </div>
  );
}

function LinkAccountSheet({
  existingLinks,
  onClose,
  stremioAccounts,
}: {
  existingLinks: StremioStremioLink[];
  onClose: () => void;
  stremioAccounts: StremioAccount[];
}) {
  const { create } = useStremioStremioLinkMutation();

  const existingPairs = useMemo(() => {
    const pairs = new Set<string>();
    for (const link of existingLinks) {
      pairs.add(`${link.account_a_id}:${link.account_b_id}`);
    }
    return pairs;
  }, [existingLinks]);

  const form = useAppForm({
    defaultValues: {
      account_a_id: "",
      account_b_id: "",
    },
    onSubmit: async ({ value }) => {
      if (value.account_a_id === value.account_b_id) {
        toast.error("Cannot link an account to itself");
        return;
      }
      if (existingPairs.has(`${value.account_a_id}:${value.account_b_id}`)) {
        toast.error("This link already exists");
        return;
      }
      await create.mutateAsync({
        account_a_id: value.account_a_id,
        account_b_id: value.account_b_id,
        sync_config: { watched: { dir: "none", ids: [] } },
      });
      toast.success("Accounts linked successfully!");
      onClose();
    },
  });

  return (
    <Form className="flex flex-col gap-4" form={form}>
      <form.AppField name="account_a_id">
        {(field) => (
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium" htmlFor={field.name}>
              Account A
            </label>
            {stremioAccounts.length === 0 ? (
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
                  {stremioAccounts.map((account) => (
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

      <form.AppField name="account_b_id">
        {(field) => (
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium" htmlFor={field.name}>
              Account B
            </label>
            {stremioAccounts.length === 0 ? (
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
                  {stremioAccounts.map((account) => (
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

      <form.AppForm>
        <form.SubmitButton
          className="w-full"
          disabled={stremioAccounts.length < 2}
        >
          Link Accounts
        </form.SubmitButton>
      </form.AppForm>
    </Form>
  );
}

function LinkCard({
  accountA,
  accountB,
  link,
}: {
  accountA?: StremioAccount;
  accountB?: StremioAccount;
  link: StremioStremioLink;
}) {
  const { remove, resetSyncState, sync, update } =
    useStremioStremioLinkMutation();

  const [isEditingItems, setIsEditingItems] = useState(false);
  const [tempIds, setTempIds] = useState<string[]>([]);

  useEffect(() => {
    setTempIds(link.sync_config.watched.ids);
  }, [link.sync_config.watched.ids]);

  const selectedWatchedSyncDirection = syncDirectionOptions.find(
    (opt) => opt.value === link.sync_config.watched.dir,
  );
  const SyncDirectionIcon = selectedWatchedSyncDirection?.icon || XCircle;

  const handleWatchedSyncDirectionChange = (value: string) => {
    toast.promise(
      update.mutateAsync({
        account_a_id: link.account_a_id,
        account_b_id: link.account_b_id,
        sync_config: {
          watched: {
            dir: value as SyncDirection,
            ids: link.sync_config.watched.ids,
          },
        },
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

  const handleSaveItems = () => {
    toast.promise(
      update.mutateAsync({
        account_a_id: link.account_a_id,
        account_b_id: link.account_b_id,
        sync_config: {
          watched: {
            dir: link.sync_config.watched.dir,
            ids: tempIds.map((id) => id),
          },
        },
      }),
      {
        error(err: APIError) {
          console.error(err);
          return {
            closeButton: true,
            message: err.message,
          };
        },
        loading: "Updating items...",
        success: {
          closeButton: true,
          message: "Items updated!",
        },
      },
    );
    setIsEditingItems(false);
  };

  const handleSync = () => {
    toast.promise(
      sync.mutateAsync({
        account_a_id: link.account_a_id,
        account_b_id: link.account_b_id,
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
        account_a_id: link.account_a_id,
        account_b_id: link.account_b_id,
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
        account_a_id: link.account_a_id,
        account_b_id: link.account_b_id,
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
              <span className="font-medium">Account A:</span>{" "}
              {accountA?.email || link.account_a_id}
            </div>
            <div>
              <span className="font-medium">Account B:</span>{" "}
              {accountB?.email || link.account_b_id}
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

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Items to Sync</label>
            {!isEditingItems && (
              <Button
                onClick={() => setIsEditingItems(true)}
                size="sm"
                variant="ghost"
              >
                Edit
              </Button>
            )}
          </div>

          {isEditingItems ? (
            <div className="flex flex-col gap-3">
              <ItemsManager ids={tempIds} onIdsChange={setTempIds} />
              <div className="flex gap-2">
                <Button
                  className="flex-1"
                  onClick={handleSaveItems}
                  size="sm"
                  variant="outline"
                >
                  Save
                </Button>
                <Button
                  className="flex-1"
                  onClick={() => setIsEditingItems(false)}
                  size="sm"
                  variant="ghost"
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <div className="text-muted-foreground text-sm">
              {link.sync_config.watched.ids.length === 0 ? (
                <span>No items selected. Click Edit to add items.</span>
              ) : (
                <span>
                  {link.sync_config.watched.ids.length} item
                  {link.sync_config.watched.ids.length !== 1 ? "s" : ""}{" "}
                  selected
                </span>
              )}
            </div>
          )}
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
          className="hidden flex-1"
          disabled={
            link.sync_config.watched.dir === "none" ||
            link.sync_config.watched.ids.length === 0 ||
            sync.isPending
          }
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
                <strong>{accountA?.email || "Account A"}</strong> and{" "}
                <strong>{accountB?.email || "Account B"}</strong>. Sync will
                stop, but your watch history won't be deleted.
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
  const links = useStremioStremioLinks();
  const stremioAccounts = useStremioAccounts();

  const [sheetOpen, setSheetOpen] = useState(false);

  const stremioAccountsById = useMemo(
    () => new Map(stremioAccounts.data?.map((acc) => [acc.id, acc])),
    [stremioAccounts.data],
  );

  const isLoading = links.isLoading || stremioAccounts.isLoading;
  const hasError = links.isError || stremioAccounts.isError;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Stremio ↔ Stremio Sync</h2>
          <p className="text-muted-foreground text-sm">
            Link Stremio accounts to sync watch history for specific titles
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
                Choose which Stremio accounts to link for sync.
              </SheetDescription>
            </SheetHeader>
            <div className="p-4">
              {stremioAccounts.data && links.data ? (
                <LinkAccountSheet
                  existingLinks={links.data}
                  onClose={() => setSheetOpen(false)}
                  stremioAccounts={stremioAccounts.data}
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
                Link your Stremio accounts to start syncing watch history
              </p>
            </div>
            {stremioAccounts.data?.length === 0 && (
              <div className="text-muted-foreground flex flex-col gap-1 text-sm">
                <div>
                  Add a{" "}
                  <Link
                    className="text-primary underline underline-offset-4"
                    to="/dash/vault/stremio-accounts"
                  >
                    Stremio account
                  </Link>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {links.data?.map((link) => (
            <LinkCard
              accountA={stremioAccountsById.get(link.account_a_id)}
              accountB={stremioAccountsById.get(link.account_b_id)}
              key={`${link.account_a_id}:${link.account_b_id}`}
              link={link}
            />
          ))}
        </div>
      )}
    </div>
  );
}
