import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { Trash2 } from "lucide-react";
import { DateTime } from "luxon";
import { toast } from "sonner";

import {
  NzbQueueItem,
  useNzbQueue,
  useNzbQueueMutation,
} from "@/api/nzb-queue";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    NzbQueue: {
      removeItem: ReturnType<typeof useNzbQueueMutation>["remove"];
    };
  }

  export interface DataTableMetaCtxKey {
    NzbQueue: NzbQueueItem;
  }
}

const col = createColumnHelper<NzbQueueItem>();

function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case "queued":
      return <Badge variant="outline">Queued</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

const columns: ColumnDef<NzbQueueItem>[] = [
  col.accessor("name", {
    cell: ({ getValue }) => {
      const name = getValue() || "<Unknown>";
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="block max-w-[300px] truncate">{name}</span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[500px] break-all">
            {name}
          </TooltipContent>
        </Tooltip>
      );
    },
    header: "Name",
  }),
  col.accessor("url", {
    cell: ({ getValue }) => {
      const url = getValue();
      const truncated = url.length > 50 ? url.substring(0, 50) + "..." : url;
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="text-muted-foreground text-xs">{truncated}</span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[500px] break-all">
            {url}
          </TooltipContent>
        </Tooltip>
      );
    },
    header: "URL",
  }),
  col.accessor("category", {
    cell: ({ getValue }) => {
      const category = getValue();
      return category || <span className="text-muted-foreground">-</span>;
    },
    header: "Category",
  }),
  col.accessor("priority", {
    header: "Priority",
  }),
  col.accessor("status", {
    cell: ({ getValue }) => <StatusBadge status={getValue()} />,
    header: "Status",
  }),
  col.accessor("error", {
    cell: ({ getValue }) => {
      const error = getValue();
      if (!error) return <span className="text-muted-foreground">-</span>;
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="text-destructive block max-w-[200px] truncate">
              {error}
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[500px] break-all">
            {error}
          </TooltipContent>
        </Tooltip>
      );
    },
    header: "Error",
  }),
  col.accessor("created_at", {
    cell: ({ getValue }) => {
      const date = DateTime.fromISO(getValue());
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Created At",
  }),
  col.display({
    cell: (c) => {
      const { removeItem } = c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button size="icon-sm" variant="ghost">
                <Trash2 className="text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Queue Item?</AlertDialogTitle>
                <AlertDialogDescription className="wrap-anywhere">
                  This will permanently delete the queue item{" "}
                  <strong>{item.name}</strong>. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={removeItem.isPending}
                    onClick={() => {
                      toast.promise(removeItem.mutateAsync(item.id), {
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

export const Route = createFileRoute("/dash/usenet/nzb-queue")({
  component: RouteComponent,
  staticData: {
    crumb: "Queue",
  },
});

function RouteComponent() {
  const nzbQueue = useNzbQueue();
  const { remove: removeItem } = useNzbQueueMutation();

  const table = useDataTable({
    columns,
    data: nzbQueue.data ?? [],
    initialState: {
      columnPinning: { left: ["name"], right: ["actions"] },
    },
    meta: {
      ctx: {
        removeItem,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">NZB Queue</h2>
      </div>

      {nzbQueue.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : nzbQueue.isError ? (
        <div className="text-sm text-red-600">Error loading NZB queue</div>
      ) : (
        <DataTable table={table} />
      )}
    </div>
  );
}
