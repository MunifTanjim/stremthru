import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { SearchIcon } from "lucide-react";
import { DateTime } from "luxon";
import prettyBytes from "pretty-bytes";
import { useMemo, useState } from "react";

import { TorrentInfoItem, useTorrentInfos } from "@/api/torrent-info";
import { DataTable } from "@/components/data-table";
import { useDataTable } from "@/components/data-table/use-data-table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export const Route = createFileRoute("/dash/torrent/torrent-info")({
  component: RouteComponent,
  staticData: {
    crumb: "Info",
  },
});

const col = createColumnHelper<TorrentInfoItem>();

const columns: ColumnDef<TorrentInfoItem>[] = [
  col.accessor("hash", {
    cell: ({ getValue }) => (
      <span className="font-mono text-xs">{getValue()}</span>
    ),
    header: "Hash",
  }),
  col.accessor("t_title", {
    cell: ({ getValue }) => (
      <Tooltip>
        <TooltipTrigger>
          <span className="inline-block max-w-sm truncate text-sm">
            {getValue()}
          </span>
        </TooltipTrigger>
        <TooltipContent>{getValue()}</TooltipContent>
      </Tooltip>
    ),
    header: "Title",
  }),
  col.accessor("imdb_id", {
    cell: ({ getValue }) => {
      const value = getValue();
      return value || <span className="text-muted-foreground">-</span>;
    },
    header: "IMDB",
  }),
  col.accessor("src", {
    header: "Source",
  }),
  col.accessor("category", {
    cell: ({ getValue }) => {
      const value = getValue();
      return value || <span className="text-muted-foreground">-</span>;
    },
    header: "Category",
  }),
  col.accessor("size", {
    cell: ({ getValue }) => {
      const value = getValue();
      if (value <= 0) {
        return <span className="text-muted-foreground">-</span>;
      }
      return prettyBytes(value);
    },
    header: "Size",
  }),
  col.accessor("created_at", {
    cell: ({ getValue }) => {
      const value = getValue();
      if (!value) {
        return <span className="text-muted-foreground">-</span>;
      }
      const date = DateTime.fromISO(value);
      return date.toLocaleString(DateTime.DATETIME_MED);
    },
    header: "Created At",
  }),
];

function RouteComponent() {
  const [input, setInput] = useState("");
  const [search, setSearch] = useState("");

  const torrentInfos = useTorrentInfos({ q: search });

  const allItems = useMemo(
    () => torrentInfos.data?.pages.flatMap((page) => page.items) ?? [],
    [torrentInfos.data],
  );

  const table = useDataTable({
    columns,
    data: allItems,
    initialState: {
      columnPinning: {
        left: ["hash"],
      },
    },
  });

  const onSearch = () => {
    setSearch(input.trim());
  };

  const onClearSearch = () => {
    setInput("");
    setSearch("");
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Torrent Info</h2>
      </div>

      <div className="flex gap-2">
        <Input
          className="max-w-sm"
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              onSearch();
            }
          }}
          placeholder="Hash / IMDB ID / Title (with basic glob * ?)"
          value={input}
        />
        <Button onClick={onSearch}>
          <SearchIcon className="mr-1 size-4" />
          Search
        </Button>
        {search && (
          <Button onClick={onClearSearch} variant="outline">
            Clear
          </Button>
        )}
      </div>

      {torrentInfos.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : torrentInfos.isError ? (
        <div className="text-sm text-red-600">Error loading torrent info</div>
      ) : (
        <>
          <DataTable table={table} />
          <div className="flex justify-center py-2">
            {torrentInfos.isFetchingNextPage ? (
              <div className="text-muted-foreground text-sm">Loading...</div>
            ) : torrentInfos.hasNextPage ? (
              <Button
                onClick={() => torrentInfos.fetchNextPage()}
                variant="outline"
              >
                Load More
              </Button>
            ) : allItems.length > 0 ? (
              <div className="text-muted-foreground text-sm">
                {allItems.length} items
              </div>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}
