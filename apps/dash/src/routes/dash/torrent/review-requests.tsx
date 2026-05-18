import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { DateTime } from "luxon";
import { useMemo, useState } from "react";

import {
  MappingTarget,
  ReviewRequest,
  ReviewStatus,
  useReviewRequests,
} from "@/api/torrent-review-requests";
import { DataTable } from "@/components/data-table";
import { useDataTable } from "@/components/data-table/use-data-table";
import { TorrentReviewRequestDetailSheet } from "@/components/torrent-review-request-detail-sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export const Route = createFileRoute("/dash/torrent/review-requests")({
  component: RouteComponent,
  staticData: {
    crumb: "Review Requests",
  },
});

const reasonLabels: Record<string, string> = {
  fake_torrent: "Fake Torrent",
  incomplete_season_pack: "Incomplete Season Pack",
  other: "Other",
  wrong_mapping: "Wrong Mapping",
};

const col = createColumnHelper<ReviewRequest>();

function getColumns(): ColumnDef<ReviewRequest>[] {
  return [
    col.accessor("hash", {
      cell: ({ getValue, row }) => {
        const hash = getValue();
        const title = row.original.hash_title;
        return (
          <div className="flex max-w-[200px] flex-col gap-0.5">
            <span className="truncate font-mono text-xs">{hash}</span>
            {title && (
              <span className="text-muted-foreground truncate text-xs">
                {title}
              </span>
            )}
          </div>
        );
      },
      header: "Hash",
    }),
    col.accessor("target", {
      cell: ({ getValue }) => (
        <Badge variant="outline">{getValue().toUpperCase()}</Badge>
      ),
      header: "Target",
    }),
    col.accessor("reason", {
      cell: ({ getValue }) => reasonLabels[getValue()] ?? getValue(),
      header: "Reason",
    }),
    col.accessor("prev_id", {
      cell: ({ getValue, row }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">—</span>;
        const titles = row.original.prev_id_titles;
        const firstTitle = titles?.[0];
        const content = (
          <div className="flex flex-col gap-0.5">
            <span className="font-mono text-xs">{value}</span>
            {firstTitle && (
              <span className="text-muted-foreground max-w-[150px] truncate text-xs">
                {firstTitle}
              </span>
            )}
          </div>
        );
        if (titles && titles.length > 1) {
          return (
            <Tooltip>
              <TooltipTrigger asChild>{content}</TooltipTrigger>
              <TooltipContent className="max-w-[300px]">
                <ul className="space-y-1 text-xs">
                  {titles.map((t, i) => (
                    <li key={i}>{t}</li>
                  ))}
                </ul>
              </TooltipContent>
            </Tooltip>
          );
        }
        return content;
      },
      header: "Prev ID",
    }),
    col.accessor("mapping_id", {
      cell: ({ getValue, row }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">—</span>;
        const titles = row.original.mapping_id_titles;
        const firstTitle = titles?.[0];
        const content = (
          <div className="flex flex-col gap-0.5">
            <span className="font-mono text-xs">{value}</span>
            {firstTitle && (
              <span className="text-muted-foreground max-w-[150px] truncate text-xs">
                {firstTitle}
              </span>
            )}
          </div>
        );
        if (titles && titles.length > 1) {
          return (
            <Tooltip>
              <TooltipTrigger asChild>{content}</TooltipTrigger>
              <TooltipContent className="max-w-[300px]">
                <ul className="space-y-1 text-xs">
                  {titles.map((t, i) => (
                    <li key={i}>{t}</li>
                  ))}
                </ul>
              </TooltipContent>
            </Tooltip>
          );
        }
        return content;
      },
      header: "Suggested ID",
    }),
    col.accessor("created_at", {
      cell: ({ getValue }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">—</span>;
        return DateTime.fromISO(value).toLocaleString(DateTime.DATETIME_MED);
      },
      header: "Created At",
    }),
    col.accessor("status", {
      cell: ({ getValue }) => {
        const status = getValue();
        return (
          <Badge variant={status === "pending" ? "secondary" : "outline"}>
            {status}
          </Badge>
        );
      },
      header: "Status",
    }),
  ];
}

function RouteComponent() {
  const [statusFilter, setStatusFilter] = useState<"all" | ReviewStatus>("all");
  const [targetFilter, setTargetFilter] = useState<"all" | MappingTarget>(
    "all",
  );
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<null | ReviewRequest>(
    null,
  );

  const reviewRequests = useReviewRequests({
    status: statusFilter === "all" ? undefined : statusFilter,
    target: targetFilter === "all" ? undefined : targetFilter,
  });

  const items = useMemo(
    () => reviewRequests.data?.pages.flatMap((page) => page.items) ?? [],
    [reviewRequests.data],
  );

  const columns = useMemo(() => getColumns(), []);

  const table = useDataTable({
    columns,
    data: items,
    getRowId: (row) => String(row.id),
  });

  function handleRowClick(item: ReviewRequest) {
    setSelectedRequest(item);
    setDetailOpen(true);
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Review Requests</h2>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4">
        <Select
          onValueChange={(v) => setStatusFilter(v as "all" | ReviewStatus)}
          value={statusFilter}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="resolved">Resolved</SelectItem>
          </SelectContent>
        </Select>

        <Select
          onValueChange={(v) => setTargetFilter(v as "all" | MappingTarget)}
          value={targetFilter}
        >
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Target" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="imdb">IMDB</SelectItem>
            <SelectItem value="anidb">AniDB</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      {reviewRequests.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : reviewRequests.isError ? (
        <div className="text-sm text-red-600">
          Error loading review requests
        </div>
      ) : (
        <>
          <DataTable
            onRowClick={(row) => handleRowClick(row.original)}
            table={table}
          />
          <div className="flex justify-center py-2">
            {reviewRequests.isFetchingNextPage ? (
              <div className="text-muted-foreground text-sm">Loading...</div>
            ) : reviewRequests.hasNextPage ? (
              <Button
                onClick={() => reviewRequests.fetchNextPage()}
                variant="outline"
              >
                Load More
              </Button>
            ) : items.length > 0 ? (
              <div className="text-muted-foreground text-sm">
                {items.length} items
              </div>
            ) : null}
          </div>
        </>
      )}

      <TorrentReviewRequestDetailSheet
        onOpenChange={setDetailOpen}
        open={detailOpen}
        request={selectedRequest}
      />
    </div>
  );
}
