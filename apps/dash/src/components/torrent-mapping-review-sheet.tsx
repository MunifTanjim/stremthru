import { useState } from "react";
import { toast } from "sonner";

import { AniDBTitle } from "@/api/anidb";
import { IMDBTitle } from "@/api/imdb";
import {
  MappingReviewRequest,
  ReviewReason,
  useSubmitMappingReview,
} from "@/api/torrent-mapping-review";

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

  const submitReview = useSubmitMappingReview();

  function resetForm() {
    setReason("");
    setCorrectedId("");
    setCorrectedTitle("");
    setComment("");
  }

  function handleIMDBSelect(title: IMDBTitle) {
    setCorrectedId(title.id);
    setCorrectedTitle(title.title);
  }

  function handleAniDBSelect(title: AniDBTitle) {
    setCorrectedId(title.id);
    setCorrectedTitle(title.title);
  }

  async function handleSubmit() {
    if (!reason) {
      toast.error("Please select a reason.");
      return;
    }

    const params: MappingReviewRequest = {
      comment: comment || undefined,
      hash,
      mapping_id: correctedId || prevId,
      prev_id: prevId,
      reason,
      target,
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
