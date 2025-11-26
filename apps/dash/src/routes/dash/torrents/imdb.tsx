import { createFileRoute } from "@tanstack/react-router";
import {
  ColumnDef,
  createColumnHelper,
  RowSelectionState,
} from "@tanstack/react-table";
import { CommandLoading } from "cmdk";
import { ExternalLinkIcon, MagnetIcon, SearchIcon } from "lucide-react";
import { useEffect, useState } from "react";
import z from "zod";

import { type IMDBTitle, useIMDBAutocomplete } from "@/api/imdb";
import {
  RequestTorrentReviewItem,
  type Torrent,
  useRequestTorrentReview,
  useTorrents,
} from "@/api/torrents";
import { DataTable } from "@/components/data-table";
import { CellSelect } from "@/components/data-table/cell-select";
import { HeaderCellSelectAll } from "@/components/data-table/header-cell-select-all";
import { useDataTable } from "@/components/data-table/use-data-table";
import { Form, useAppForm } from "@/components/form";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemHeader,
  ItemTitle,
} from "@/components/ui/item";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
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
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useSearchParams } from "@/hooks/use-search-params";
import { cn } from "@/lib/utils";

const SERIES_TYPES = ["tvMiniSeries", "tvSeries", "tvShort", "tvSpecial"];

function isSeries(type?: string): boolean {
  return type ? SERIES_TYPES.includes(type) : false;
}

function parseSid(sid?: string): null | { episode: number; season: number } {
  if (!sid) return null;
  const parts = sid.split(":");
  if (parts.length !== 3) return null;
  const season = parseInt(parts[1], 10);
  const episode = parseInt(parts[2], 10);
  if (isNaN(season) || isNaN(episode)) return null;
  return { episode, season };
}

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    Torrent: {
      setSelectedTorrent: (torrent: Torrent) => void;
    };
  }

  export interface DataTableMetaCtxKey {
    Torrent: Torrent;
  }
}

const columnHelper = createColumnHelper<Torrent>();

const torrentColumns: ColumnDef<Torrent>[] = [
  columnHelper.display({
    cell: CellSelect,
    enableHiding: false,
    enableSorting: false,
    header: HeaderCellSelectAll,
    id: "select",
  }),
  columnHelper.accessor("private", {
    cell: ({ getValue }) => (getValue() ? "🔑" : ""),
    header: "",
  }),
  columnHelper.accessor("hash", {
    cell: ({ getValue }) => (
      <div className="max-w-md truncate font-medium">{getValue()}</div>
    ),
    header: "Hash",
  }),
  columnHelper.accessor("name", {
    cell: ({ getValue }) => (
      <div className="max-w-md truncate font-medium">{getValue()}</div>
    ),
    header: "Name",
  }),
  columnHelper.accessor("size", {
    cell: ({ getValue }) => <div>{getValue()}</div>,
    header: "Size",
  }),
  columnHelper.accessor("seeders", {
    cell: ({ getValue }) => {
      const value = getValue();
      return (
        <div className={cn("font-medium", value && "text-green-600")}>
          {value || "-"}
        </div>
      );
    },
    header: "Seeders",
  }),
];

export const Route = createFileRoute("/dash/torrents/imdb")({
  component: RouteComponent,
  staticData: {
    crumb: "IMDB",
  },
  validateSearch: z.object({
    imdbid: z.string().optional(),
  }),
});

function RouteComponent() {
  const [searchParams, setSearchParams] = useSearchParams(Route.fullPath);

  const [searchOpen, setSearchOpen] = useState(Boolean(searchParams.imdbid));
  const [_searchQuery, setSearchQuery] = useState(searchParams.imdbid);
  const searchQuery = useDebouncedValue(_searchQuery, 300);

  const [selectedTitle, setSelectedTitle] = useState<IMDBTitle | null>(null);
  const autocompleteResults = useIMDBAutocomplete(searchQuery);
  const torrentsResults = useTorrents(selectedTitle?.id);

  const [requestReviewItems, setRequestReviewItems] = useState<
    Record<string, RequestTorrentReviewItem>
  >({});

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [reviewModalOpen, setReviewModalOpen] = useState(false);
  const [selectedTorrentHash, setSelectedTorrentHash] = useState<string>("");

  const requestReviewMutation = useRequestTorrentReview();

  const reviewForm = useAppForm({
    defaultValues: {
      comment: "",
      imdb_id: selectedTitle?.id || "",
      prev_imdb_id: selectedTitle?.id || "",
      reason: "wrong_mapping" as RequestTorrentReviewItem["reason"],
    },
    onSubmit: async ({ value }) => {
      // Save current form values to the review items state
      if (selectedTorrentHash && requestReviewItems[selectedTorrentHash]) {
        updateReviewItem(selectedTorrentHash, value);
      }

      // Submit all review items
      await requestReviewMutation.mutateAsync({
        items: Object.values(requestReviewItems),
      });

      // Close modal and reset state
      setReviewModalOpen(false);
      setRequestReviewItems({});
      setRowSelection({});
    },
    validators: {
      onChange: z.object({
        comment: z.string().optional(),
        imdb_id: z.string().optional(),
        prev_imdb_id: z.string().optional(),
        reason: z.enum([
          "fake_torrent",
          "incomplete_season_pack",
          "other",
          "wrong_mapping",
          "wrong_title",
        ]),
      }),
    },
  });

  const table = useDataTable({
    columns: torrentColumns,
    data: torrentsResults.data ?? [],
    getRowId: (row) => row.hash,
    onRowSelectionChange: setRowSelection,
    state: {
      rowSelection,
    },
  });

  const selectedTorrents =
    torrentsResults.data?.filter((torrent) => rowSelection[torrent.hash]) ?? [];

  const handleOpenReviewModal = () => {
    if (selectedTorrents.length === 0) return;

    // Initialize review items for selected torrents that don't have one yet
    const newReviewItems = { ...requestReviewItems };
    selectedTorrents.forEach((torrent) => {
      if (!newReviewItems[torrent.hash]) {
        newReviewItems[torrent.hash] = {
          files:
            isSeries(selectedTitle?.type) && torrent.files
              ? torrent.files.map((f) => {
                  const parsed = parseSid(f.sid);
                  return {
                    ep: parsed?.episode ?? 0,
                    path: f.path,
                    prev_ep: parsed?.episode ?? 0,
                    prev_s: parsed?.season ?? 0,
                    s: parsed?.season ?? 0,
                  };
                })
              : undefined,
          hash: torrent.hash,
          imdb_id: selectedTitle?.id,
          reason: "wrong_mapping",
        };
      }
    });
    setRequestReviewItems(newReviewItems);

    // Set the first selected torrent as the active one
    setSelectedTorrentHash(selectedTorrents[0].hash);
    setReviewModalOpen(true);
  };

  const updateReviewItem = (
    hash: string,
    updates: Partial<RequestTorrentReviewItem>,
  ) => {
    setRequestReviewItems((prev) => ({
      ...prev,
      [hash]: {
        ...prev[hash],
        ...updates,
      },
    }));
  };

  const updateFileReview = (
    fileIndex: number,
    field: "ep" | "s",
    value: number,
  ) => {
    if (!selectedTorrentHash || !requestReviewItems[selectedTorrentHash]?.files)
      return;

    setRequestReviewItems((prev) => ({
      ...prev,
      [selectedTorrentHash]: {
        ...prev[selectedTorrentHash],
        files: prev[selectedTorrentHash].files?.map((f, idx) =>
          idx === fileIndex ? { ...f, [field]: value } : f,
        ),
      },
    }));
  };

  const currentReviewItem = selectedTorrentHash
    ? requestReviewItems[selectedTorrentHash]
    : null;

  const currentTorrent = selectedTorrents.find(
    (t) => t.hash === selectedTorrentHash,
  );

  // Sync form values when selected torrent changes
  useEffect(() => {
    if (currentReviewItem) {
      reviewForm.setFieldValue("reason", currentReviewItem.reason);
      reviewForm.setFieldValue("imdb_id", currentReviewItem.imdb_id || "");
      reviewForm.setFieldValue(
        "prev_imdb_id",
        currentReviewItem.prev_imdb_id || selectedTitle?.id || "",
      );
      reviewForm.setFieldValue("comment", currentReviewItem.comment || "");
    }
  }, [selectedTorrentHash, currentReviewItem, selectedTitle?.id, reviewForm]);

  // Save form values when switching torrents
  const handleTorrentChange = (newHash: string) => {
    // Save current form values to the review items state before switching
    if (selectedTorrentHash && requestReviewItems[selectedTorrentHash]) {
      const currentValues = reviewForm.state.values;
      updateReviewItem(selectedTorrentHash, currentValues);
    }
    setSelectedTorrentHash(newHash);
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Search</CardTitle>
          <CardDescription>Search for Movies or TV Shows</CardDescription>
        </CardHeader>
        <CardContent>
          <Popover onOpenChange={setSearchOpen} open={searchOpen}>
            <PopoverTrigger asChild>
              <Button
                aria-expanded={searchOpen}
                className="w-[300px] justify-between"
                role="combobox"
                variant="outline"
              >
                {selectedTitle ? selectedTitle?.title : "Search..."}
                <SearchIcon className="opacity-50" />
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-[300px] p-0">
              <Command shouldFilter={false}>
                <CommandInput
                  onValueChange={setSearchQuery}
                  placeholder="Search IMDB titles..."
                  value={_searchQuery}
                />
                <CommandList>
                  <CommandEmpty>IMDB Titles</CommandEmpty>
                  {autocompleteResults.isLoading && _searchQuery && (
                    <CommandLoading>Searching...</CommandLoading>
                  )}
                  {autocompleteResults.data?.map((title) => (
                    <CommandItem
                      key={title.id}
                      onSelect={async () => {
                        setSelectedTitle(title);
                        setSearchParams((prev) => ({
                          ...prev,
                          imdbid: title.id,
                        }));
                        setRequestReviewItems({});
                        setSearchQuery(title.title);
                        setSearchOpen(false);
                      }}
                      value={title.id}
                    >
                      <Item className="w-full p-0" size="sm">
                        <ItemHeader className="text-muted-foreground flex justify-between text-xs">
                          <div>{title.type}</div>
                          <div>{title.id}</div>
                        </ItemHeader>
                        <ItemContent>
                          <ItemTitle>{title.title}</ItemTitle>
                          <ItemDescription>
                            <span className="text-muted-foreground text-xs">
                              {title.year}
                            </span>
                          </ItemDescription>
                        </ItemContent>
                      </Item>
                    </CommandItem>
                  ))}
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        </CardContent>
      </Card>

      {selectedTitle && (
        <Card>
          <CardHeader>
            <CardTitle>Available Torrents</CardTitle>
            <CardDescription>
              {selectedTitle.title} ({selectedTitle.year}) - {selectedTitle.id}
            </CardDescription>
            <CardAction>
              <Button
                disabled={selectedTorrents.length === 0}
                onClick={handleOpenReviewModal}
                variant="destructive"
              >
                {Object.keys(rowSelection).length}
                <MagnetIcon />
                Review
              </Button>
            </CardAction>
          </CardHeader>
          <CardContent>
            {torrentsResults.isLoading ? (
              <div className="py-6 text-center text-sm">
                Loading torrents...
              </div>
            ) : (
              <DataTable table={table} />
            )}
          </CardContent>
        </Card>
      )}

      <Sheet onOpenChange={setReviewModalOpen} open={reviewModalOpen}>
        <SheetContent className="flex max-h-screen w-full flex-col overflow-hidden sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Review Torrents</SheetTitle>
            <SheetDescription>
              Fill in the review details for each selected torrent
              {selectedTorrents.length > 1 && (
                <Field>
                  <FieldLabel htmlFor="torrent-select">
                    Select Torrent
                  </FieldLabel>
                  <Select
                    onValueChange={handleTorrentChange}
                    value={selectedTorrentHash}
                  >
                    <SelectTrigger className="max-w-full" id="torrent-select">
                      <SelectValue placeholder="Select a torrent" />
                    </SelectTrigger>
                    <SelectContent>
                      {selectedTorrents.map((torrent) => (
                        <SelectItem key={torrent.hash} value={torrent.hash}>
                          <div className="max-w-md truncate">
                            {torrent.name}
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              )}
              {currentTorrent && (
                <div className="bg-muted mt-2 rounded-md p-3">
                  <div className="text-muted-foreground mb-1 text-xs">
                    {currentTorrent.hash}
                  </div>
                  <div className="text-sm font-medium">
                    {currentTorrent.name}
                  </div>
                </div>
              )}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className="shrink grow overflow-y-hidden px-4 pb-4">
            {currentReviewItem && currentTorrent && (
              <Form className="flex shrink flex-col gap-4" form={reviewForm}>
                <reviewForm.AppField name="reason">
                  {(field) => (
                    <field.Select
                      label="Reason"
                      options={[
                        { label: "Wrong Mapping", value: "wrong_mapping" },
                        { label: "Wrong Title", value: "wrong_title" },
                        { label: "Fake Torrent", value: "fake_torrent" },
                        {
                          label: "Incomplete Season Pack",
                          value: "incomplete_season_pack",
                        },
                        { label: "Other", value: "other" },
                      ]}
                      required
                    />
                  )}
                </reviewForm.AppField>

                <reviewForm.AppField name="prev_imdb_id">
                  {(field) => (
                    <field.Input
                      label={
                        <div className="flex w-full flex-row items-center justify-between">
                          <div>Previous IMDB ID</div>
                          <div>
                            <Button asChild size="icon-sm" variant="link">
                              <a
                                href={`https://imdb.com/title/${field.state.value}`}
                                rel="noreferrer noopener"
                                target="_blank"
                              >
                                <ExternalLinkIcon />
                              </a>
                            </Button>
                          </div>
                        </div>
                      }
                      readOnly
                    />
                  )}
                </reviewForm.AppField>

                <reviewForm.AppField name="imdb_id">
                  {(field) => (
                    <field.Input
                      label={
                        <div className="flex w-full flex-row items-center justify-between">
                          <div>IMDB ID</div>
                          <div>
                            <Button asChild size="icon-sm" variant="link">
                              <a
                                href={`https://imdb.com/title/${field.state.value}`}
                                rel="noreferrer noopener"
                                target="_blank"
                              >
                                <ExternalLinkIcon />
                              </a>
                            </Button>
                          </div>
                        </div>
                      }
                      placeholder="tt0000000"
                    />
                  )}
                </reviewForm.AppField>

                {isSeries(selectedTitle?.type) &&
                  currentTorrent?.files &&
                  currentTorrent.files.length > 0 && (
                    <Field>
                      <FieldLabel>Episode Mapping</FieldLabel>
                      <div className="max-h-64 overflow-y-auto rounded-md border">
                        <table className="w-full text-sm">
                          <thead className="bg-muted sticky top-0">
                            <tr>
                              <th className="p-2 text-left">File</th>
                              <th className="w-20 p-2 text-center">Prev S</th>
                              <th className="w-20 p-2 text-center">Prev Ep</th>
                              <th className="w-20 p-2 text-center">S</th>
                              <th className="w-20 p-2 text-center">Ep</th>
                            </tr>
                          </thead>
                          <tbody>
                            {currentTorrent.files.map((file, idx) => {
                              const reviewFile =
                                currentReviewItem?.files?.[idx];
                              if (!reviewFile) return null;
                              return (
                                <tr className="border-t" key={file.path}>
                                  <td
                                    className="max-w-xs truncate p-2"
                                    title={file.path}
                                  >
                                    {file.name}
                                  </td>
                                  <td className="text-muted-foreground p-2 text-center">
                                    {reviewFile.prev_s}
                                  </td>
                                  <td className="text-muted-foreground p-2 text-center">
                                    {reviewFile.prev_ep}
                                  </td>
                                  <td className="p-2">
                                    <Input
                                      className="w-16 text-center"
                                      min={0}
                                      onChange={(e) =>
                                        updateFileReview(
                                          idx,
                                          "s",
                                          parseInt(e.target.value, 10) || 0,
                                        )
                                      }
                                      type="number"
                                      value={reviewFile.s}
                                    />
                                  </td>
                                  <td className="p-2">
                                    <Input
                                      className="w-16 text-center"
                                      min={0}
                                      onChange={(e) =>
                                        updateFileReview(
                                          idx,
                                          "ep",
                                          parseInt(e.target.value, 10) || 0,
                                        )
                                      }
                                      type="number"
                                      value={reviewFile.ep}
                                    />
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    </Field>
                  )}

                <reviewForm.AppField name="comment">
                  {(field) => (
                    <field.Input
                      label="Comment"
                      placeholder="Additional details..."
                    />
                  )}
                </reviewForm.AppField>
              </Form>
            )}

            <ScrollBar orientation="vertical" />
          </ScrollArea>

          <SheetFooter className="border-t">
            <div className="flex flex-wrap items-center justify-between">
              <div className="text-muted-foreground text-sm">
                {selectedTorrents.length} torrent
                {selectedTorrents.length !== 1 ? "s" : ""} selected
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={() => setReviewModalOpen(false)}
                  type="button"
                  variant="outline"
                >
                  Cancel
                </Button>
                <Button variant="destructive">Submit Review</Button>
              </div>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}
