import { useState } from "react";
import { toast } from "sonner";

import { AniDBTitle } from "@/api/anidb";
import { IMDBTitle } from "@/api/imdb";
import {
  AniDBMappingPreview,
  ReviewReason,
  ReviewRequest,
  usePreviewMapping,
  useRejectReviewRequest,
  useResolveReviewRequest,
} from "@/api/torrent-review-requests";

import { AniDBSearch } from "./anidb-search";
import { IMDBSearch } from "./imdb-search";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { ScrollArea } from "./ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "./ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";

type MappingPreviewTableProps = {
  mappings: AniDBMappingPreview[];
  onAdd: () => void;
  onChange: (index: number, field: keyof AniDBMappingPreview, value: string | number) => void;
  onRemove: (index: number) => void;
};

function MappingPreviewTable({ mappings, onAdd, onChange, onRemove }: MappingPreviewTableProps) {
  return (
    <div className="flex flex-col gap-2">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Type</TableHead>
            <TableHead>Season</TableHead>
            <TableHead>Ep Start</TableHead>
            <TableHead>Ep End</TableHead>
            <TableHead className="w-16">
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {mappings.map((mapping, i) => (
            <TableRow key={i}>
              <TableCell>
                <Select
                  onValueChange={(val) => onChange(i, "s_type", val)}
                  value={mapping.s_type}
                >
                  <SelectTrigger aria-label="Season type" className="w-20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="abs">abs</SelectItem>
                    <SelectItem value="tv">tv</SelectItem>
                    <SelectItem value="ani">ani</SelectItem>
                  </SelectContent>
                </Select>
              </TableCell>
              <TableCell>
                <Input
                  aria-label="Season"
                  className="w-16"
                  min={0}
                  onChange={(e) => onChange(i, "s", Number(e.target.value))}
                  type="number"
                  value={mapping.s}
                />
              </TableCell>
              <TableCell>
                <Input
                  aria-label="Episode start"
                  className="w-20"
                  min={0}
                  onChange={(e) => onChange(i, "ep_start", Number(e.target.value))}
                  type="number"
                  value={mapping.ep_start}
                />
              </TableCell>
              <TableCell>
                <Input
                  aria-label="Episode end"
                  className="w-20"
                  min={0}
                  onChange={(e) => onChange(i, "ep_end", Number(e.target.value))}
                  type="number"
                  value={mapping.ep_end}
                />
              </TableCell>
              <TableCell>
                <Button
                  onClick={() => onRemove(i)}
                  size="sm"
                  type="button"
                  variant="destructive"
                >
                  Delete
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Button onClick={onAdd} size="sm" type="button" variant="outline">
        + Add Row
      </Button>
    </div>
  );
}

type TorrentReviewRequestDetailSheetProps = {
  onOpenChange: (open: boolean) => void;
  open: boolean;
  request: null | ReviewRequest;
};

const reasonLabels: Record<ReviewReason, string> = {
  fake_torrent: "Fake Torrent",
  incomplete_season_pack: "Incomplete Season Pack",
  other: "Other",
  wrong_mapping: "Wrong Mapping",
};

export function TorrentReviewRequestDetailSheet({
  onOpenChange,
  open,
  request,
}: TorrentReviewRequestDetailSheetProps) {
  const [finalMappingId, setFinalMappingId] = useState("");
  const [finalMappingTitle, setFinalMappingTitle] = useState("");
  const [previewMappings, setPreviewMappings] = useState<AniDBMappingPreview[] | null>(null);
  const [isPreviewMode, setIsPreviewMode] = useState(false);

  const resolveReviewRequest = useResolveReviewRequest();
  const previewMutation = usePreviewMapping();
  const rejectMutation = useRejectReviewRequest();

  function handleIMDBSelect(title: IMDBTitle) {
    setFinalMappingId(title.id);
    setFinalMappingTitle(title.title);
  }

  function handleAniDBSelect(title: AniDBTitle) {
    setFinalMappingId(title.id);
    setFinalMappingTitle(title.title);
  }

  async function handlePreview() {
    if (!request || !finalMappingId) return;

    try {
      const result = await previewMutation.mutateAsync({
        id: request.id,
        params: { anidb_id: finalMappingId },
      });
      setPreviewMappings(result.mappings);
      setIsPreviewMode(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to preview mappings.";
      toast.error(message);
    }
  }

  function handleCancelPreview() {
    setPreviewMappings(null);
    setIsPreviewMode(false);
  }

  async function handleReject() {
    if (!request) return;

    if (previewMappings && previewMappings.length > 0) {
      if (!window.confirm("This will delete all mappings and mark as unmappable. Continue?")) {
        return;
      }
    }

    try {
      await rejectMutation.mutateAsync(request.id);
      toast.success("Review request rejected!");
      handleOpenChange(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to reject review.";
      toast.error(message);
    }
  }

  function handleMappingChange(index: number, field: keyof AniDBMappingPreview, value: string | number) {
    if (!previewMappings) return;
    const updated = [...previewMappings];
    updated[index] = { ...updated[index], [field]: value };
    setPreviewMappings(updated);
  }

  function handleAddMapping() {
    const newMapping: AniDBMappingPreview = {
      tid: finalMappingId,
      s_type: "tv",
      s: 1,
      ep_start: 1,
      ep_end: 1,
    };
    setPreviewMappings([...(previewMappings || []), newMapping]);
  }

  function handleRemoveMapping(index: number) {
    if (!previewMappings) return;
    setPreviewMappings(previewMappings.filter((_, i) => i !== index));
  }

  function handleOpenChange(value: boolean) {
    if (!value) {
      setFinalMappingId("");
      setFinalMappingTitle("");
      setPreviewMappings(null);
      setIsPreviewMode(false);
    }
    onOpenChange(value);
  }

  async function handleResolve() {
    if (!request) return;

    // Validate mappings for AniDB
    if (request.target === "anidb" && previewMappings) {
      if (previewMappings.length === 0) {
        toast.error("No mappings to apply. Add at least one mapping or reject.");
        return;
      }

      for (const m of previewMappings) {
        if (m.ep_start > m.ep_end) {
          toast.error("Invalid episode range: start cannot be greater than end.");
          return;
        }
        if (m.s < 0 || m.ep_start < 0 || m.ep_end < 0) {
          toast.error("Season and episode numbers must be non-negative.");
          return;
        }
      }
    }

    try {
      if (request.target === "anidb" && previewMappings) {
        await resolveReviewRequest.mutateAsync({
          id: request.id,
          mappings: previewMappings,
        });
      } else {
        const mapping_id = finalMappingId || request.mapping_id || undefined;
        await resolveReviewRequest.mutateAsync({
          id: request.id,
          mapping_id,
        });
      }
      toast.success("Review request resolved successfully!");
      handleOpenChange(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to resolve review request.";
      toast.error(message);
    }
  }

  const searchTriggerLabel = finalMappingTitle
    ? finalMappingTitle
    : finalMappingId
      ? finalMappingId
      : request?.mapping_id
        ? request.mapping_id
        : "Search...";

  return (
    <Sheet onOpenChange={handleOpenChange} open={open}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Review Request Details</SheetTitle>
        </SheetHeader>

        <ScrollArea className="overflow-hidden">
          <div className="flex flex-col gap-4">
            {request && (
              <div className="flex flex-col gap-4 p-4">
                <DetailRow label="Hash">
                  <div className="flex flex-col gap-1">
                    <span className="break-all font-mono text-xs">
                      {request.hash}
                    </span>
                    {request.hash_title && (
                      <span className="text-muted-foreground text-sm">
                        {request.hash_title}
                      </span>
                    )}
                  </div>
                </DetailRow>

                <DetailRow label="Target">
                  <Badge variant="outline">
                    {request.target.toUpperCase()}
                  </Badge>
                </DetailRow>

                <DetailRow label="Reason">
                  {reasonLabels[request.reason]}
                </DetailRow>

                {request.comment && (
                  <DetailRow label="Comment">
                    <span className="text-muted-foreground">
                      {request.comment}
                    </span>
                  </DetailRow>
                )}

                <DetailRow label="Current Mapping ID">
                  <div className="flex flex-col gap-1">
                    <span className="font-mono">{request.prev_id || "—"}</span>
                    {request.prev_id_titles &&
                      request.prev_id_titles.length > 0 && (
                        <ul className="text-muted-foreground space-y-0.5 text-sm">
                          {request.prev_id_titles.map((title, i) => (
                            <li key={i}>{title}</li>
                          ))}
                        </ul>
                      )}
                  </div>
                </DetailRow>

                <DetailRow label="Suggested Mapping ID">
                  <div className="flex flex-col gap-1">
                    <span className="font-mono">
                      {request.mapping_id || "—"}
                    </span>
                    {request.mapping_id_titles &&
                      request.mapping_id_titles.length > 0 && (
                        <ul className="text-muted-foreground space-y-0.5 text-sm">
                          {request.mapping_id_titles.map((title, i) => (
                            <li key={i}>{title}</li>
                          ))}
                        </ul>
                      )}
                  </div>
                </DetailRow>

                <DetailRow label="Status">
                  <Badge
                    variant={
                      request.status === "pending" ? "secondary" : "outline"
                    }
                  >
                    {request.status}
                  </Badge>
                </DetailRow>

                <DetailRow label="Created At">
                  {new Date(request.created_at).toLocaleString()}
                </DetailRow>

                {request.files && request.files.length > 0 && (
                  <DetailRow label="File Corrections">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Path</TableHead>
                          <TableHead>Prev S</TableHead>
                          <TableHead>Prev E</TableHead>
                          <TableHead>S</TableHead>
                          <TableHead>E</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {request.files.map((file, i) => (
                          <TableRow key={i}>
                            <TableCell className="max-w-[200px] truncate font-mono text-xs">
                              {file.path}
                            </TableCell>
                            <TableCell>{file.prev_season}</TableCell>
                            <TableCell>{file.prev_episode}</TableCell>
                            <TableCell>{file.season}</TableCell>
                            <TableCell>{file.episode}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </DetailRow>
                )}

                {request.suggested_mappings && request.suggested_mappings.length > 0 && (
                  <DetailRow label="Suggested Mappings">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Type</TableHead>
                          <TableHead>Season</TableHead>
                          <TableHead>Ep Start</TableHead>
                          <TableHead>Ep End</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {request.suggested_mappings.map((m, i) => (
                          <TableRow key={i}>
                            <TableCell>{m.s_type}</TableCell>
                            <TableCell>{m.s}</TableCell>
                            <TableCell>{m.ep_start}</TableCell>
                            <TableCell>{m.ep_end}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </DetailRow>
                )}

                {request.status === "pending" && (
                  <div className="flex flex-col gap-1.5">
                    {!isPreviewMode ? (
                      <>
                        <Label>Final Mapping ID</Label>
                        {request.target === "imdb" ? (
                          <IMDBSearch
                            onSelect={handleIMDBSelect}
                            triggerLabel={searchTriggerLabel}
                          />
                        ) : (
                          <>
                            <AniDBSearch
                              onSelect={handleAniDBSelect}
                              triggerLabel={searchTriggerLabel}
                            />
                            {finalMappingId && (
                              <Button
                                disabled={previewMutation.isPending}
                                onClick={handlePreview}
                                size="sm"
                                type="button"
                                variant="outline"
                              >
                                {previewMutation.isPending
                                  ? "Loading..."
                                  : "Preview Mappings"}
                              </Button>
                            )}
                          </>
                        )}
                        {finalMappingId && (
                          <p className="text-muted-foreground text-xs">
                            {finalMappingId}
                          </p>
                        )}
                      </>
                    ) : (
                      <>
                        <Label>Preview Mappings</Label>
                        <MappingPreviewTable
                          mappings={previewMappings ?? []}
                          onAdd={handleAddMapping}
                          onChange={handleMappingChange}
                          onRemove={handleRemoveMapping}
                        />
                      </>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </ScrollArea>

        <SheetFooter>
          <Button
            disabled={resolveReviewRequest.isPending}
            onClick={() => handleOpenChange(false)}
            variant="outline"
          >
            Cancel
          </Button>
          {isPreviewMode && (
            <Button
              disabled={resolveReviewRequest.isPending}
              onClick={handleCancelPreview}
              variant="outline"
            >
              Back
            </Button>
          )}
          {request?.status === "pending" && (
            <Button
              disabled={rejectMutation.isPending}
              onClick={handleReject}
              variant="destructive"
            >
              {rejectMutation.isPending ? "Rejecting..." : "Reject"}
            </Button>
          )}
          {request?.status === "pending" &&
            (request.target === "imdb" || isPreviewMode) && (
              <Button
                disabled={resolveReviewRequest.isPending}
                onClick={handleResolve}
              >
                {resolveReviewRequest.isPending ? "Applying..." : "Apply"}
              </Button>
            )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function DetailRow({
  children,
  label,
}: {
  children: React.ReactNode;
  label: string;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-muted-foreground text-xs">{label}</Label>
      <div className="text-sm">{children}</div>
    </div>
  );
}
