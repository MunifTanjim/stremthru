import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import { MoreHorizontal, RefreshCwIcon, SearchIcon } from "lucide-react";
import { DateTime } from "luxon";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { AniDBTitle } from "@/api/anidb";
import { IMDBTitle } from "@/api/imdb";
import {
  AniDBMappingItem,
  IMDBMappingItem,
  MappingMode,
  ReprocessItem,
  useAniDBMappings,
  useIMDBMappings,
  useReprocessTorrents,
} from "@/api/torrent-info";
import { AniDBSearch } from "@/components/anidb-search";
import { DataTable } from "@/components/data-table";
import { useDataTable } from "@/components/data-table/use-data-table";
import { IMDBSearch } from "@/components/imdb-search";
import { TorrentMappingReviewSheet } from "@/components/torrent-mapping-review-sheet";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export const Route = createFileRoute("/dash/torrent/info")({
  component: RouteComponent,
  staticData: {
    crumb: "Info",
  },
});

const SERIES_TYPES = ["tvMiniSeries", "tvSeries"];

// IMDB columns definition
const imdbCol = createColumnHelper<IMDBMappingItem>();
function getImdbColumns(
  onReview: (item: IMDBMappingItem) => void,
): ColumnDef<IMDBMappingItem>[] {
  return [
    imdbCol.display({
      cell: ({ row }) => (
        <Checkbox
          aria-label="Select row"
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
        />
      ),
      enableHiding: false,
      enableSorting: false,
      header: ({ table }) => (
        <Checkbox
          aria-label="Select all"
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        />
      ),
      id: "select",
      size: 32,
    }),
    imdbCol.accessor("hash", {
      cell: ({ getValue }) => (
        <span className="font-mono text-xs">{getValue()}</span>
      ),
      header: "Hash",
    }),
    imdbCol.accessor("t_title", {
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
      header: "Torrent Title",
    }),
    imdbCol.accessor("imdb_id", {
      header: "IMDB ID",
    }),
    imdbCol.accessor("imdb_title", {
      cell: ({ row }) => {
        const { imdb_title, imdb_type, imdb_year } = row.original;
        if (!imdb_title) {
          return <span className="text-muted-foreground">-</span>;
        }
        return (
          <span>
            {imdb_title} ({imdb_year}){" "}
            <span className="text-muted-foreground text-xs">[{imdb_type}]</span>
          </span>
        );
      },
      header: "IMDB Title",
    }),
    imdbCol.accessor("mapped_at", {
      cell: ({ getValue }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">-</span>;
        return DateTime.fromISO(value).toLocaleString(DateTime.DATETIME_MED);
      },
      header: "Mapped At",
    }),
    imdbCol.display({
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="ghost">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={() => onReview(row.original)}>
              Request Review
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      enableHiding: false,
      enableSorting: false,
      id: "actions",
      size: 40,
    }),
  ];
}

// AniDB columns definition
const anidbCol = createColumnHelper<AniDBMappingItem>();
function getAnidbColumns(
  onReview: (item: AniDBMappingItem) => void,
): ColumnDef<AniDBMappingItem>[] {
  return [
    {
      cell: ({ row }) => (
        <Checkbox
          aria-label="Select row"
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
        />
      ),
      enableHiding: false,
      enableSorting: false,
      header: ({ table }) => (
        <Checkbox
          aria-label="Select all"
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        />
      ),
      id: "select",
    },
    anidbCol.accessor("hash", {
      cell: ({ getValue }) => (
        <span className="font-mono text-xs">{getValue()}</span>
      ),
      header: "Hash",
    }),
    anidbCol.accessor("t_title", {
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
      header: "Torrent Title",
    }),
    anidbCol.accessor("anidb_id", {
      header: "AniDB ID",
    }),
    anidbCol.accessor("anidb_title", {
      cell: ({ getValue }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">-</span>;
        return value;
      },
      header: "AniDB Title",
    }),
    anidbCol.accessor("season_type", {
      cell: ({ row }) => {
        const { ep_end, ep_start, season, season_type } = row.original;
        if (!season_type) {
          return <span className="text-muted-foreground">-</span>;
        }
        const epRange =
          ep_start === ep_end ? `Ep ${ep_start}` : `Ep ${ep_start}-${ep_end}`;
        return (
          <span className="text-sm">
            S{season} ({season_type}) - {epRange}
          </span>
        );
      },
      header: "Season/Episode",
    }),
    anidbCol.accessor("mapped_at", {
      cell: ({ getValue }) => {
        const value = getValue();
        if (!value) return <span className="text-muted-foreground">-</span>;
        return DateTime.fromISO(value).toLocaleString(DateTime.DATETIME_MED);
      },
      header: "Mapped At",
    }),
    anidbCol.display({
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="ghost">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={() => onReview(row.original)}>
              Request Review
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      enableHiding: false,
      enableSorting: false,
      id: "actions",
      size: 40,
    }),
  ];
}

function RouteComponent() {
  const [tab, setTab] = useState<"anidb" | "imdb">("imdb");
  const [mode, setMode] = useState<MappingMode>("by-id");
  const [showUnmapped, setShowUnmapped] = useState(false);
  const [input, setInput] = useState("");
  const [search, setSearch] = useState("");
  const [selectedTitle, setSelectedTitle] = useState<IMDBTitle | null>(null);
  const [selectedAniDBTitle, setSelectedAniDBTitle] =
    useState<AniDBTitle | null>(null);
  const [season, setSeason] = useState("");
  const [episode, setEpisode] = useState("");
  const [anidbEpisode, setAnidbEpisode] = useState("");
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const [reviewSheetOpen, setReviewSheetOpen] = useState(false);
  const [reviewItem, setReviewItem] = useState<null | {
    hash: string;
    prevId: string;
    target: "anidb" | "imdb";
  }>(null);

  const reprocessMutation = useReprocessTorrents();

  const handleReviewItem = (
    item: AniDBMappingItem | IMDBMappingItem,
    target: "anidb" | "imdb",
  ) => {
    const prevId =
      target === "imdb"
        ? (item as IMDBMappingItem).imdb_id
        : (item as AniDBMappingItem).anidb_id;
    setReviewItem({ hash: item.hash, prevId, target });
    setReviewSheetOpen(true);
  };

  const isSeries = selectedTitle && SERIES_TYPES.includes(selectedTitle.type);
  const isAniDBSeries =
    selectedAniDBTitle && selectedAniDBTitle.type !== "MOVIE";

  const imdbMappings = useIMDBMappings({
    mode,
    q: tab === "imdb" ? search : "",
    unmapped: mode === "by-title" ? showUnmapped : undefined,
  });
  const anidbMappings = useAniDBMappings({
    mode,
    q: tab === "anidb" ? search : "",
    unmapped: mode === "by-title" ? showUnmapped : undefined,
  });

  const imdbItems = useMemo(
    () => imdbMappings.data?.pages.flatMap((page) => page.items) ?? [],
    [imdbMappings.data],
  );
  const anidbItems = useMemo(
    () => anidbMappings.data?.pages.flatMap((page) => page.items) ?? [],
    [anidbMappings.data],
  );

  const imdbColumns = useMemo(
    () => getImdbColumns((item) => handleReviewItem(item, "imdb")),

    [],
  );
  const anidbColumns = useMemo(
    () => getAnidbColumns((item) => handleReviewItem(item, "anidb")),

    [],
  );

  const imdbTable = useDataTable({
    columns: imdbColumns,
    data: imdbItems,
    getRowId: (row) => `${row.imdb_id ?? ""}:${row.hash}`,
    initialState: { columnPinning: { left: ["select", "hash"] } },
    onRowSelectionChange: setRowSelection,
    state: { rowSelection },
  });
  const anidbTable = useDataTable({
    columns: anidbColumns,
    data: anidbItems,
    getRowId: (row) =>
      `${row.anidb_id ?? ""}:${row.hash}:${row.season_type ?? ""}:${row.season ?? 0}`,
    initialState: { columnPinning: { left: ["select", "hash"] } },
    onRowSelectionChange: setRowSelection,
    state: { rowSelection },
  });

  const currentQuery = tab === "imdb" ? imdbMappings : anidbMappings;
  const currentItems = tab === "imdb" ? imdbItems : anidbItems;

  const onSearch = () => setSearch(input.trim());
  const onClearSearch = () => {
    setInput("");
    setSearch("");
    setSelectedTitle(null);
    setSelectedAniDBTitle(null);
    setSeason("");
    setEpisode("");
    setAnidbEpisode("");
    setRowSelection({});
  };

  const selectedItems = (() => {
    const seen = new Set<string>();
    const items: ReprocessItem[] = [];

    for (const key of Object.keys(rowSelection)) {
      if (!rowSelection[key]) continue;

      const parts = key.split(":");
      if (tab === "imdb") {
        // IMDB: tid:hash
        items.push({ tid: parts[0], hash: parts[1] });
      } else {
        // AniDB: tid:hash:s_type:s - extract only tid:hash, dedupe
        const dedupeKey = `${parts[0]}:${parts[1]}`;
        if (!seen.has(dedupeKey)) {
          seen.add(dedupeKey);
          items.push({ tid: parts[0], hash: parts[1] });
        }
      }
    }
    return items;
  })();

  const handleReprocess = (items: ReprocessItem[]) => {
    reprocessMutation.mutate(
      { items, targets: [tab] },
      {
        onError: (error) => {
          toast.error(`Error: ${error.message}`);
        },
        onSuccess: (data) => {
          setRowSelection({});
          if (data.mode === "sync") {
            toast.success(
              `Reprocessed: Parsed ${data.parsed}, Mapped IMDB ${data.mapped?.imdb ?? 0}, AniDB ${data.mapped?.anidb ?? 0}`,
            );
          } else {
            toast.success(`Queued ${data.queued} torrents for reprocessing`);
          }
        },
      },
    );
  };

  const placeholder =
    mode === "by-id"
      ? tab === "imdb"
        ? "IMDB ID (tt1234567) or title"
        : "AniDB ID or title"
      : "Torrent hash or title (with glob * ?)";

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Torrent Info</h2>
      </div>

      {/* Tab selection and mode selection */}
      <div className="flex flex-wrap items-center gap-4">
        <ButtonGroup>
          <Button
            onClick={() => {
              setTab("imdb");
              onClearSearch();
            }}
            size="sm"
            variant={tab === "imdb" ? "default" : "outline"}
          >
            IMDB
          </Button>
          <Button
            onClick={() => {
              setTab("anidb");
              onClearSearch();
            }}
            size="sm"
            variant={tab === "anidb" ? "default" : "outline"}
          >
            AniDB
          </Button>
        </ButtonGroup>

        <Select onValueChange={(v) => setMode(v as MappingMode)} value={mode}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="by-id">
              By {tab === "imdb" ? "IMDB" : "AniDB"}
            </SelectItem>
            <SelectItem value="by-title">By Title</SelectItem>
          </SelectContent>
        </Select>

        {mode === "by-title" && (
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-sm">Mapped</span>
            <Switch checked={showUnmapped} onCheckedChange={setShowUnmapped} />
            <span className="text-muted-foreground text-sm">Unmapped</span>
          </div>
        )}
      </div>

      {/* Search */}
      <div className="flex w-full flex-wrap gap-2">
        {mode === "by-id" ? (
          tab === "imdb" ? (
            <>
              <div className="w-64">
                <IMDBSearch
                  onSelect={(title) => {
                    setSelectedTitle(title);
                    setSeason("");
                    setEpisode("");
                    if (!SERIES_TYPES.includes(title.type)) {
                      setSearch(title.id);
                    }
                  }}
                  triggerLabel={
                    selectedTitle
                      ? `${selectedTitle.title} (${selectedTitle.id})`
                      : undefined
                  }
                />
              </div>
              {isSeries && (
                <>
                  <Input
                    className="w-24"
                    onChange={(e) => setSeason(e.target.value)}
                    placeholder="Season"
                    type="number"
                    value={season}
                  />
                  <Input
                    className="w-24"
                    onChange={(e) => setEpisode(e.target.value)}
                    placeholder="Episode"
                    type="number"
                    value={episode}
                  />
                  <Button
                    onClick={() => {
                      let stremId = selectedTitle?.id || "";
                      if (season) {
                        stremId += `:${season}`;
                        if (episode) {
                          stremId += `:${episode}`;
                        }
                      }
                      setSearch(stremId);
                    }}
                  >
                    <SearchIcon className="mr-1 size-4" />
                    Search
                  </Button>
                </>
              )}
            </>
          ) : (
            <>
              <div className="w-64">
                <AniDBSearch
                  onSelect={(title) => {
                    setSelectedAniDBTitle(title);
                    setAnidbEpisode("");
                    if (title.type === "MOVIE") {
                      setSearch(`anidb:${title.id}`);
                    }
                  }}
                  triggerLabel={
                    selectedAniDBTitle
                      ? `${selectedAniDBTitle.title} (${selectedAniDBTitle.id})`
                      : undefined
                  }
                />
              </div>
              {isAniDBSeries && (
                <>
                  <Input
                    className="w-24"
                    onChange={(e) => setAnidbEpisode(e.target.value)}
                    placeholder="Episode"
                    type="number"
                    value={anidbEpisode}
                  />
                  <Button
                    onClick={() => {
                      let searchId = `anidb:${selectedAniDBTitle?.id}`;
                      if (anidbEpisode) {
                        searchId += `:${anidbEpisode}`;
                      }
                      setSearch(searchId);
                    }}
                  >
                    <SearchIcon className="mr-1 size-4" />
                    Search
                  </Button>
                </>
              )}
            </>
          )
        ) : (
          <>
            <Input
              className="max-w-md"
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") onSearch();
              }}
              placeholder={placeholder}
              value={input}
            />
            <Button onClick={onSearch}>
              <SearchIcon className="mr-1 size-4" />
              Search
            </Button>
          </>
        )}

        {search && (
          <Button onClick={onClearSearch} variant="outline">
            Clear
          </Button>
        )}

        {/* Actions */}
        {selectedItems.length > 0 && (
          <div className="ml-auto flex items-center gap-2">
            <span className="text-muted-foreground text-sm">
              {selectedItems.length} selected
            </span>
            <Button
              disabled={reprocessMutation.isPending}
              onClick={() => handleReprocess(selectedItems)}
              variant="outline"
            >
              <RefreshCwIcon
                className={`size-4 ${reprocessMutation.isPending ? "animate-spin" : ""}`}
              />
              Reprocess
            </Button>
            <Button onClick={() => setRowSelection({})} variant="ghost">
              Clear
            </Button>
          </div>
        )}
      </div>

      {reviewItem && (
        <TorrentMappingReviewSheet
          hash={reviewItem.hash}
          onOpenChange={setReviewSheetOpen}
          open={reviewSheetOpen}
          prevId={reviewItem.prevId}
          target={reviewItem.target}
        />
      )}

      {/* Results */}
      {currentQuery.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : currentQuery.isError ? (
        <div className="text-sm text-red-600">Error loading mappings</div>
      ) : (
        <>
          {tab === "imdb" ? (
            <DataTable table={imdbTable} />
          ) : (
            <DataTable table={anidbTable} />
          )}
          <div className="flex justify-center py-2">
            {currentQuery.isFetchingNextPage ? (
              <div className="text-muted-foreground text-sm">Loading...</div>
            ) : currentQuery.hasNextPage ? (
              <Button
                onClick={() => currentQuery.fetchNextPage()}
                variant="outline"
              >
                Load More
              </Button>
            ) : currentItems.length > 0 ? (
              <div className="text-muted-foreground text-sm">
                {currentItems.length} items
              </div>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}
