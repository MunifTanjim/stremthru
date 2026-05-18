import { useState } from "react";
import { toast } from "sonner";

import { AniDBTitle } from "@/api/anidb";
import { IMDBTitle } from "@/api/imdb";
import {
  MappingReviewRequest,
  ReviewReason,
  SuggestedMapping,
  useSubmitMappingReview,
} from "@/api/torrent-review-requests";

import { AniDBSearch } from "./anidb-search";
import { IMDBSearch } from "./imdb-search";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
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
import { Textarea } from "./ui/textarea";

type TorrentMappingReviewSheetProps = {
  hash: string;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  prevId: string;
  target: "anidb" | "imdb";
};

const reasonLabels: Record<ReviewReason, string> = {
  fake_torrent: "Fake Torrent",
  incomplete_season_pack: "Incomplete Season Pack",
  other: "Other",
  wrong_mapping: "Wrong Mapping",
};

const reviewReasons: ReviewReason[] = [
  "wrong_mapping",
  "fake_torrent",
  "incomplete_season_pack",
  "other",
];

export function TorrentMappingReviewSheet({
  hash,
  onOpenChange,
  open,
  prevId,
  target,
}: TorrentMappingReviewSheetProps) {
  const [reason, setReason] = useState<"" | ReviewReason>("");
  const [correctedId, setCorrectedId] = useState("");
  const [correctedTitle, setCorrectedTitle] = useState("");
  const [comment, setComment] = useState("");
  const [suggestedMappings, setSuggestedMappings] = useState<SuggestedMapping[]>([]);

  const submitReview = useSubmitMappingReview();

  function resetForm() {
    setReason("");
    setCorrectedId("");
    setCorrectedTitle("");
    setComment("");
    setSuggestedMappings([]);
  }

  function handleIMDBSelect(title: IMDBTitle) {
    setCorrectedId(title.id);
    setCorrectedTitle(title.title);
  }

  function handleAniDBSelect(title: AniDBTitle) {
    setCorrectedId(title.id);
    setCorrectedTitle(title.title);
  }

  function handleMappingChange(index: number, field: keyof SuggestedMapping, value: string | number) {
    const updated = [...suggestedMappings];
    updated[index] = { ...updated[index], [field]: value };
    setSuggestedMappings(updated);
  }

  function handleAddMapping() {
    setSuggestedMappings([
      ...suggestedMappings,
      { s_type: "tv", s: 1, ep_start: 1, ep_end: 1 },
    ]);
  }

  function handleRemoveMapping(index: number) {
    setSuggestedMappings(suggestedMappings.filter((_, i) => i !== index));
  }

  async function handleSubmit() {
    if (!reason) {
      toast.error("Please select a reason.");
      return;
    }

    // Validate: if AniDB with corrected ID, require at least one mapping
    if (target === "anidb" && correctedId && suggestedMappings.length === 0) {
      toast.error("Please add at least one suggested mapping.");
      return;
    }

    // Validate mapping values
    for (const m of suggestedMappings) {
      if (m.ep_start > m.ep_end) {
        toast.error("Invalid episode range: start cannot be greater than end.");
        return;
      }
      if (m.s < 0 || m.ep_start < 0 || m.ep_end < 0) {
        toast.error("Season and episode numbers must be non-negative.");
        return;
      }
    }

    const params: MappingReviewRequest = {
      comment: comment || undefined,
      hash,
      mapping_id: correctedId || prevId,
      prev_id: prevId,
      reason,
      target,
      suggested_mappings: target === "anidb" && suggestedMappings.length > 0 ? suggestedMappings : undefined,
    };

    try {
      await submitReview.mutateAsync(params);
      toast.success("Review submitted successfully!");
      resetForm();
      onOpenChange(false);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to submit review.";
      toast.error(message);
    }
  }

  const searchTriggerLabel = correctedTitle
    ? correctedTitle
    : correctedId
      ? correctedId
      : "Search...";

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Request Mapping Review</SheetTitle>
        </SheetHeader>

        <div className="flex flex-col gap-4 p-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="review-reason">Reason</Label>
            <Select
              onValueChange={(val) => setReason(val as ReviewReason)}
              value={reason}
            >
              <SelectTrigger className="w-full" id="review-reason">
                <SelectValue placeholder="Select a reason..." />
              </SelectTrigger>
              <SelectContent>
                {reviewReasons.map((r) => (
                  <SelectItem key={r} value={r}>
                    {reasonLabels[r]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="current-id">Current ID</Label>
            <Input disabled id="current-id" readOnly value={prevId} />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Corrected ID</Label>
            {target === "imdb" ? (
              <IMDBSearch
                onSelect={handleIMDBSelect}
                triggerLabel={searchTriggerLabel}
              />
            ) : (
              <AniDBSearch
                onSelect={handleAniDBSelect}
                triggerLabel={searchTriggerLabel}
              />
            )}
            {correctedId && (
              <p className="text-muted-foreground text-xs">{correctedId}</p>
            )}
          </div>

          {target === "anidb" && correctedId && (
            <div className="flex flex-col gap-2">
              <Label>Suggested Mappings</Label>
              {suggestedMappings.length > 0 && (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Type</TableHead>
                      <TableHead>Season</TableHead>
                      <TableHead>Ep Start</TableHead>
                      <TableHead>Ep End</TableHead>
                      <TableHead><span className="sr-only">Actions</span></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {suggestedMappings.map((mapping, index) => (
                      <TableRow key={index}>
                        <TableCell>
                          <Select
                            value={mapping.s_type}
                            onValueChange={(v) => handleMappingChange(index, "s_type", v)}
                          >
                            <SelectTrigger className="h-8 w-20" aria-label="Season type">
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
                            type="number"
                            className="h-8 w-16"
                            value={mapping.s}
                            onChange={(e) => handleMappingChange(index, "s", parseInt(e.target.value) || 0)}
                            aria-label="Season"
                          />
                        </TableCell>
                        <TableCell>
                          <Input
                            type="number"
                            className="h-8 w-16"
                            value={mapping.ep_start}
                            onChange={(e) => handleMappingChange(index, "ep_start", parseInt(e.target.value) || 0)}
                            aria-label="Episode start"
                          />
                        </TableCell>
                        <TableCell>
                          <Input
                            type="number"
                            className="h-8 w-16"
                            value={mapping.ep_end}
                            onChange={(e) => handleMappingChange(index, "ep_end", parseInt(e.target.value) || 0)}
                            aria-label="Episode end"
                          />
                        </TableCell>
                        <TableCell>
                          <Button variant="ghost" size="sm" onClick={() => handleRemoveMapping(index)}>
                            ✕
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
              <Button variant="outline" size="sm" onClick={handleAddMapping}>
                + Add Mapping
              </Button>
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="review-comment">Comment (optional)</Label>
            <Textarea
              id="review-comment"
              onChange={(e) => setComment(e.target.value)}
              placeholder="Provide additional context..."
              value={comment}
            />
          </div>
        </div>

        <SheetFooter>
          <Button
            disabled={submitReview.isPending}
            onClick={() => onOpenChange(false)}
            variant="outline"
          >
            Cancel
          </Button>
          <Button disabled={submitReview.isPending} onClick={handleSubmit}>
            {submitReview.isPending ? "Submitting..." : "Submit Review"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
