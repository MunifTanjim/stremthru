import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef, createColumnHelper } from "@tanstack/react-table";
import {
  ChevronDown,
  ChevronRight,
  Download,
  ExternalLink,
  Eye,
  FileIcon,
  FolderArchive,
  PackageOpen,
  RefreshCw,
  Trash2,
  Video,
} from "lucide-react";
import { DateTime } from "luxon";
import prettyBytes from "pretty-bytes";
import { useState } from "react";
import { toast } from "sonner";

import {
  NZBContentFile,
  NZBInfoItem,
  useNzbInfo,
  useNzbInfoMutation,
} from "@/api/nzb-info";
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { APIError } from "@/lib/api";
import { cn } from "@/lib/utils";

declare module "@/components/data-table" {
  export interface DataTableMetaCtx {
    NzbInfo: {
      removeItem: ReturnType<typeof useNzbInfoMutation>["remove"];
      requeueItem: ReturnType<typeof useNzbInfoMutation>["requeue"];
      setDetailItem: (item: null | NZBInfoItem) => void;
    };
  }

  export interface DataTableMetaCtxKey {
    NzbInfo: NZBInfoItem;
  }
}

const col = createColumnHelper<NZBInfoItem>();

const columns: ColumnDef<NZBInfoItem>[] = [
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
  col.accessor("size", {
    cell: ({ getValue }) => prettyBytes(getValue()),
    header: "Size",
  }),
  col.accessor("file_count", {
    cell: ({ getValue }) => getValue(),
    header: "Files",
  }),
  col.accessor("streamable", {
    cell: ({ getValue }) =>
      getValue() ? (
        <Badge className="bg-green-600" variant="default">
          Yes
        </Badge>
      ) : (
        <Badge variant="destructive">No</Badge>
      ),
    header: "Streamable",
  }),
  col.accessor("cached", {
    cell: ({ getValue }) =>
      getValue() ? (
        <Badge className="bg-green-600" variant="default">
          Yes
        </Badge>
      ) : (
        <Badge variant="destructive">No</Badge>
      ),
    header: "Cached",
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
      const { removeItem, requeueItem, setDetailItem } =
        c.table.options.meta!.ctx;
      const item = c.row.original;
      return (
        <div className="flex gap-1">
          <Button
            onClick={() => setDetailItem(item)}
            size="icon-sm"
            variant="ghost"
          >
            <Eye />
          </Button>
          <Button
            disabled={!item.cached}
            onClick={() =>
              window.open(`/dash/api/usenet/nzb/${item.id}/xml`, "_blank")
            }
            size="icon-sm"
            variant="ghost"
          >
            <ExternalLink />
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button disabled={!item.url} size="icon-sm" variant="ghost">
                <RefreshCw />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Re-queue NZB for processing?
                </AlertDialogTitle>
                <AlertDialogDescription className="wrap-anywhere">
                  This will re-process <strong>{item.name}</strong>.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction asChild>
                  <Button
                    disabled={requeueItem.isPending}
                    onClick={() => {
                      toast.promise(requeueItem.mutateAsync(item.id), {
                        error(err: APIError) {
                          console.error(err);
                          return {
                            closeButton: true,
                            message: err.message,
                          };
                        },
                        loading: "Re-queuing...",
                        success: {
                          closeButton: true,
                          message: "Re-queued successfully!",
                        },
                      });
                    }}
                  >
                    Re-queue
                  </Button>
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button size="icon-sm" variant="ghost">
                <Trash2 className="text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete NZB Info?</AlertDialogTitle>
                <AlertDialogDescription className="wrap-anywhere">
                  This will permanently delete the NZB info for{" "}
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

function ContentFileIcon({ isPack, type }: { isPack: boolean; type: string }) {
  switch (type) {
    case "archive":
      return isPack ? (
        <PackageOpen className="size-4 text-amber-700" />
      ) : (
        <FolderArchive className="size-4 text-amber-500" />
      );
    case "video":
      return <Video className="size-4 text-blue-500" />;
    default:
      return <FileIcon className="text-muted-foreground size-4" />;
  }
}

function ContentFileNode({
  depth,
  file,
  nzbId,
  parentPath,
}: {
  depth: number;
  file: NZBContentFile;
  nzbId: string;
  parentPath?: string;
}) {
  const [expanded, setExpanded] = useState(false);
  const hasChildren = Boolean(file.files && file.files.length > 0);
  const isPack = Boolean(
    file.type === "archive" && file.parts && file.parts.length > 0,
  );
  const filePath = parentPath
    ? parentPath + "::/" + file.name
    : "/" + file.name;

  return (
    <>
      <Item
        size="sm"
        style={{ marginLeft: `${depth * 20}px` }}
        variant="outline"
      >
        <ItemMedia>
          <div className="flex h-10 flex-col justify-between">
            <ContentFileIcon isPack={isPack} type={file.type} />
            {hasChildren ? (
              <button
                className="flex size-4 shrink-0 items-center justify-center"
                onClick={() => setExpanded(!expanded)}
                type="button"
              >
                {expanded ? (
                  <ChevronDown className="size-3" />
                ) : (
                  <ChevronRight className="size-3" />
                )}
              </button>
            ) : (
              <span className="size-4 shrink-0" />
            )}
          </div>
        </ItemMedia>
        <ItemContent className="overflow-hidden">
          <ItemTitle className="flex w-full justify-between">
            <Tooltip>
              <TooltipTrigger className="flex-1 truncate text-left">
                <span>{file.name}</span>
              </TooltipTrigger>
              <TooltipContent className="max-w-[400px] break-all">
                {file.name}
              </TooltipContent>
            </Tooltip>
            <Badge className="text-xs" variant="outline">
              {file.type}
            </Badge>
          </ItemTitle>
          <ItemDescription asChild>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-muted-foreground text-xs">
                  {prettyBytes(file.size)}
                </span>
                <Badge
                  className={cn(
                    "py-0",
                    file.streamable ? "bg-green-600" : "line-through",
                  )}
                  variant={file.streamable ? "default" : "destructive"}
                >
                  Streamable
                </Badge>
                {file.errors?.map((error) => (
                  <Badge className="py-0" key={error} variant="destructive">
                    {error === "article_not_found"
                      ? "Article Not Found"
                      : error === "open_failed"
                        ? "Open Failed"
                        : error}
                  </Badge>
                ))}
              </div>
              <div className="ml-auto">
                {!hasChildren && (
                  <Button
                    disabled={!file.streamable}
                    onClick={() =>
                      window.open(
                        `/dash/api/usenet/nzb/${nzbId}/download${filePath}`,
                        "_blank",
                      )
                    }
                    size="icon-sm"
                    variant="ghost"
                  >
                    <Download className="size-3" />
                  </Button>
                )}
              </div>
            </div>
          </ItemDescription>
        </ItemContent>
      </Item>
      {hasChildren && expanded && (
        <ContentFileTree
          depth={depth + 1}
          files={file.files!}
          nzbId={nzbId}
          parentPath={filePath}
        />
      )}
    </>
  );
}

function ContentFileTree({
  depth = 0,
  files,
  nzbId,
  parentPath,
}: {
  depth?: number;
  files: NZBContentFile[];
  nzbId: string;
  parentPath?: string;
}) {
  return (
    <div className="space-y-1">
      {files.map((file) => (
        <ItemGroup className="space-y-1" key={`${depth}-${file.name}`}>
          <ContentFileNode
            depth={depth}
            file={file}
            nzbId={nzbId}
            parentPath={parentPath}
          />
          {file.parts && file.parts.length > 0 && (
            <ContentFileTree
              depth={depth}
              files={file.parts}
              nzbId={nzbId}
              parentPath={parentPath}
            />
          )}
        </ItemGroup>
      ))}
    </div>
  );
}

function NzbInfoDetailDialog({
  item,
  onClose,
}: {
  item: null | NZBInfoItem;
  onClose: () => void;
}) {
  return (
    <Dialog onOpenChange={(open) => !open && onClose()} open={Boolean(item)}>
      <DialogContent className="max-h-[80vh] max-w-2xl overflow-y-auto">
        <DialogHeader className="max-w-full">
          <DialogTitle className="break-all">{item?.name}</DialogTitle>
        </DialogHeader>
        {item && (
          <div className="space-y-4 overflow-hidden">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <div className="text-muted-foreground font-medium">Hash</div>
                <div className="mt-1 break-all font-mono text-xs">
                  {item.hash}
                </div>
              </div>
              <div>
                <div className="text-muted-foreground font-medium">Size</div>
                <div className="mt-1">{prettyBytes(item.size)}</div>
              </div>
              <div>
                <div className="text-muted-foreground font-medium">
                  Streamable
                </div>
                <div className="mt-1">
                  {item.streamable ? (
                    <Badge className="bg-green-600" variant="default">
                      Yes
                    </Badge>
                  ) : (
                    <Badge variant="destructive">No</Badge>
                  )}
                </div>
              </div>
              <div>
                <div className="text-muted-foreground font-medium">Cached</div>
                <div className="mt-1">
                  {item.cached ? (
                    <Badge className="bg-green-600" variant="default">
                      Yes
                    </Badge>
                  ) : (
                    <Badge variant="destructive">No</Badge>
                  )}
                </div>
              </div>
              <div>
                <div className="text-muted-foreground font-medium">
                  Password
                </div>
                <div className="mt-1 break-all">
                  {item.password || (
                    <span className="text-muted-foreground">-</span>
                  )}
                </div>
              </div>
              <div>
                <div className="text-muted-foreground font-medium">User</div>
                <div className="mt-1">{item.user}</div>
              </div>
              {item.url && (
                <div className="col-span-2">
                  <div className="text-muted-foreground font-medium">URL</div>
                  <div className="mt-1 break-all text-xs">{item.url}</div>
                </div>
              )}
              <div>
                <div className="text-muted-foreground font-medium">
                  File Count
                </div>
                <div className="mt-1">{item.file_count}</div>
              </div>
            </div>
            {item.files && item.files.length > 0 && (
              <div>
                <div className="text-muted-foreground mb-2 text-sm font-medium">
                  Files
                </div>
                <div className="rounded-md border p-1">
                  <ContentFileTree files={item.files} nzbId={item.id} />
                </div>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

export const Route = createFileRoute("/dash/usenet/nzb")({
  component: RouteComponent,
  staticData: {
    crumb: "NZB",
  },
});

function RouteComponent() {
  const nzbInfo = useNzbInfo();
  const { remove: removeItem, requeue: requeueItem } = useNzbInfoMutation();
  const [detailItem, setDetailItem] = useState<null | NZBInfoItem>(null);

  const table = useDataTable({
    columns,
    data: nzbInfo.data ?? [],
    initialState: {
      columnPinning: { left: ["name"], right: ["actions"] },
    },
    meta: {
      ctx: {
        removeItem,
        requeueItem,
        setDetailItem,
      },
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">NZB Info</h2>
      </div>

      {nzbInfo.isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : nzbInfo.isError ? (
        <div className="text-sm text-red-600">Error loading NZB info</div>
      ) : (
        <DataTable table={table} />
      )}

      <NzbInfoDetailDialog
        item={detailItem}
        onClose={() => setDetailItem(null)}
      />
    </div>
  );
}
