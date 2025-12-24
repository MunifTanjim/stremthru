import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { AlertCircle, Plus, SearchIcon } from "lucide-react";
import { DateTime } from "luxon";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import {
  TorznabIndexerSyncInfo,
  useTorznabIndexerSyncInfoMutation,
  useTorznabIndexerSyncInfos,
} from "@/api/torznab-indexer-syncinfo";
import { useTorznabIndexers } from "@/api/vault-torznab-indexer";
import { DataTable } from "@/components/data-table";
import { DataTablePagination } from "@/components/data-table/pagination";
import { useDataTable } from "@/components/data-table/use-data-table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    TorznabIndexerSyncInfo: {
      indexerNameById: Map<string, string>;
    };
  }

  export interface DataTableMetaCtxKey {
    TorznabIndexerSyncInfo: TorznabIndexerSyncInfo;
  }
}

export const Route = createFileRoute("/dash/torrents/indexers-sync")({
  component: RouteComponent,
  staticData: {
    crumb: "Indexers Sync",
  },
});

const col = createColumnHelper<TorznabIndexerSyncInfo>();

const columns: ColumnDef<TorznabIndexerSyncInfo>[] = [
  col.display({
    cell: ({ row, table }) => {
      const { indexerNameById } = table.options.meta!.ctx;
      const { id, type } = row.original;
      const compositeId = `${type}:${id}`;
      const name = indexerNameById.get(compositeId);
      return name || compositeId;
    },
    header: "Indexer",
    id: "indexer",
  }),
  col.accessor("sid", {
    cell: ({ getValue }) => {
      const sid = getValue();
      return <span className="font-mono text-xs">{sid}</span>;
    },
    header: "Strem Id",
  }),
  col.accessor("queued_at", {
    cell: ({ getValue }) => {
      const value = getValue();
      if (!value) {
        return <span className="text-muted-foreground">-</span>;
      }
      const date = DateTime.fromISO(value);
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Queued At",
  }),
  col.accessor("synced_at", {
    cell: ({ getValue }) => {
      const value = getValue();
      if (!value) {
        return <span className="text-muted-foreground">-</span>;
      }
      const date = DateTime.fromISO(value);
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Synced At",
  }),
  col.accessor("result_count", {
    cell: ({ getValue }) => {
      const value = getValue();
      if (value === null) {
        return <span className="text-muted-foreground">-</span>;
      }
      return value.toLocaleString();
    },
    header: "Results",
  }),
  col.accessor("error", {
    cell: ({ getValue }) => {
      const error = getValue();
      if (!error) {
        return null;
      }
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button size="icon-sm" variant="ghost">
              <AlertCircle className="text-destructive" />
            </Button>
          </TooltipTrigger>
          <TooltipContent className="max-w-md">
            <p className="text-sm">{error}</p>
          </TooltipContent>
        </Tooltip>
      );
    },
    header: "Error",
  }),
];

function RouteComponent() {
  const [searchInput, setSearchInput] = useState("");
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 10 });
  const [queueDialogOpen, setQueueDialogOpen] = useState(false);
  const [queueInput, setQueueInput] = useState("");

  const indexers = useTorznabIndexers();
  const indexerNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const indexer of indexers.data ?? []) {
      map.set(indexer.id, indexer.name);
    }
    return map;
  }, [indexers.data]);

  const [searchSId, setSearchSId] = useState("");
  const syncInfos = useTorznabIndexerSyncInfos({
    limit: pagination.pageSize,
    offset: pagination.pageIndex * pagination.pageSize,
    sid: searchSId,
  });

  const { queue: queueMutation } = useTorznabIndexerSyncInfoMutation();

  const table = useDataTable({
    columns,
    data: syncInfos.data?.items ?? [],
    manualPagination: true,
    meta: {
      ctx: {
        indexerNameById,
      },
    },
    onPaginationChange: (updater) => {
      console.log("updater", updater);
      if (typeof updater === "function") {
        const newState = updater(pagination);
        setPagination(newState);
      }
    },
    pageCount: Math.ceil(
      (syncInfos.data?.total_count ?? 0) / pagination.pageSize,
    ),
    state: {
      pagination,
    },
  });

  const onSearch = () => {
    setSearchSId(searchInput.trim());
    setPagination((p) => ({ ...p, pageIndex: 0 }));
  };

  const onClearSearch = () => {
    setSearchInput("");
    setSearchSId("");
    setPagination((p) => ({ ...p, pageIndex: 0 }));
  };

  const onQueue = () => {
    toast.promise(queueMutation.mutateAsync(queueInput.trim()), {
      error(err: APIError) {
        console.error(err);
        return {
          closeButton: true,
          message: err.message,
        };
      },
      loading: "Queueing...",
      success: () => {
        setQueueInput("");
        setQueueDialogOpen(false);
        return {
          closeButton: true,
          message: `Queued Successfully!`,
        };
      },
    });
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Indexers Sync Info</h2>
      </div>

      <div className="flex gap-2">
        <Input
          className="max-w-sm"
          onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              onSearch();
            }
          }}
          placeholder="Search by Strem Id"
          value={searchInput}
        />
        <Button onClick={onSearch}>
          <SearchIcon className="mr-1 size-4" />
          Search
        </Button>
        {searchSId && (
          <Button onClick={onClearSearch} variant="outline">
            Clear
          </Button>
        )}
        <Button onClick={() => setQueueDialogOpen(true)} variant="outline">
          <Plus className="mr-1 size-4" />
          Queue
        </Button>
      </div>

      <Dialog onOpenChange={setQueueDialogOpen} open={queueDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Queue Strem ID</DialogTitle>
            <DialogDescription>
              Enter an IMDb ID to queue for syncing.
            </DialogDescription>
          </DialogHeader>
          <Input
            onChange={(e) => setQueueInput(e.target.value)}
            placeholder="e.g., tt1234567"
            value={queueInput}
          />
          <DialogFooter>
            <Button
              disabled={!queueInput.startsWith("tt") || queueMutation.isPending}
              onClick={onQueue}
            >
              Queue
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {syncInfos.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : syncInfos.isError ? (
        <div className="text-sm text-red-600">
          Error loading indexers sync info
        </div>
      ) : (
        <>
          <DataTable table={table} />
          <DataTablePagination table={table} />
        </>
      )}
    </div>
  );
}
