# AniDB Review Resolution - Hybrid Preview Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable reviewers to preview, edit, and confirm AniDB mappings before applying resolution, with reject capability.

**Architecture:** Add preview endpoint that runs mapper without saving, modify resolve to accept explicit mappings array for AniDB, add reject endpoint for both targets. Frontend adds editable preview table with add/remove row capabilities.

**Tech Stack:** Go backend, React/TypeScript frontend, TanStack Query for data fetching

**Spec:** `docs/superpowers/specs/2026-05-16-anidb-review-resolution-design.md`

---

## File Structure

### Backend Changes
| File | Change | Responsibility |
|------|--------|----------------|
| `internal/torrent_mapping_review/db.go` | Modify | Add `rejected` status, add `Reject()` function |
| `internal/dash/api/torrent_review_requests.go` | Modify | Add preview handler, reject handler, modify resolve for AniDB |

### Frontend Changes
| File | Change | Responsibility |
|------|--------|----------------|
| `apps/dash/src/api/torrent-review-requests.ts` | Modify | Add preview/reject types and API calls |
| `apps/dash/src/components/torrent-review-request-detail-sheet.tsx` | Modify | Add preview UI, editable table, reject flow |

---

## Task 1: Add Rejected Status to Database Layer

**Files:**
- Modify: `internal/torrent_mapping_review/db.go:21-25` (ReviewStatus const block)
- Modify: `internal/torrent_mapping_review/db.go:272-279` (add Reject function)

- [ ] **Step 1.1: Add rejected status constant**

In `internal/torrent_mapping_review/db.go`, add to the ReviewStatus const block:

```go
const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusResolved ReviewStatus = "resolved"
	ReviewStatusRejected ReviewStatus = "rejected"
)
```

- [ ] **Step 1.2: Add Reject database function**

After the `Resolve` function, add:

```go
func Reject(id int) error {
	query := fmt.Sprintf(
		"UPDATE %s SET %s = ?, %s = ? WHERE %s = ?",
		TableName, Column.Status, Column.ResolvedAt, Column.Id,
	)
	_, err := db.Exec(query, ReviewStatusRejected, time.Now().UTC(), id)
	if err != nil {
		log.Error("failed to reject review", "error", err)
	}
	return err
}
```

- [ ] **Step 1.3: Verify build**

Run: `go build ./internal/torrent_mapping_review/...`
Expected: No errors

- [ ] **Step 1.4: Commit**

```bash
git add internal/torrent_mapping_review/db.go
git commit -m "feat(review): add rejected status and Reject db function"
```

---

## Task 2: Add Preview Endpoint to Backend

**Files:**
- Modify: `internal/dash/api/torrent_review_requests.go`

- [ ] **Step 2.1: Add preview request/response types**

After the existing type definitions (around line 30), add:

```go
type PreviewMappingRequest struct {
	AniDBId string `json:"anidb_id"`
}

type PreviewMappingResponse struct {
	Mappings    []AniDBMappingPreview `json:"mappings"`
	AniDBTitles []string              `json:"anidb_titles"`
}

type AniDBMappingPreview struct {
	TId     string `json:"tid"`
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}
```

- [ ] **Step 2.2: Add preview handler function**

Add after the resolve handler:

```go
func handlePreviewTorrentReviewMapping(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ErrorBadRequest(r, "invalid id").Send(w, r)
		return
	}

	review, err := torrent_mapping_review.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if review == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if review.Status != torrent_mapping_review.ReviewStatusPending {
		ErrorBadRequest(r, "review is not pending").Send(w, r)
		return
	}

	if review.Target != torrent_mapping_review.MappingTargetAniDB {
		ErrorBadRequest(r, "preview only supported for anidb").Send(w, r)
		return
	}

	var payload PreviewMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	if payload.AniDBId == "" {
		ErrorBadRequest(r, "anidb_id required").Send(w, r)
		return
	}

	tInfo, err := torrent_info.GetByHash(review.Hash)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if tInfo == nil {
		ErrorNotFound(r, "torrent info not found").Send(w, r)
		return
	}

	items := worker.MapTorrentToAniDB(review.Hash, *tInfo, func(msg string, e error, args ...any) {
		if e != nil {
			log.Error(msg, append([]any{"error", e}, args...)...)
		} else {
			log.Debug(msg, args...)
		}
	})

	mappings := make([]AniDBMappingPreview, len(items))
	for i, item := range items {
		mappings[i] = AniDBMappingPreview{
			TId:     payload.AniDBId,
			SType:   string(item.SeasonType),
			S:       item.Season,
			EpStart: item.EpisodeStart,
			EpEnd:   item.EpisodeEnd,
		}
	}

	titles := []string{}
	anidbTitles, err := anidb.GetTitlesByTId(payload.AniDBId)
	if err == nil {
		for _, t := range anidbTitles {
			titles = append(titles, t.Value)
		}
	}

	SendData(w, r, 200, PreviewMappingResponse{
		Mappings:    mappings,
		AniDBTitles: titles,
	})
}
```

- [ ] **Step 2.3: Register preview endpoint**

In `AddTorrentReviewRequestsEndpoints` function, add:

```go
router.HandleFunc("/torrent/review-requests/{id}/preview", authed(handlePreviewTorrentReviewMapping))
```

- [ ] **Step 2.4: Verify build**

Run: `go build ./internal/dash/api/...`
Expected: No errors

- [ ] **Step 2.5: Commit**

```bash
git add internal/dash/api/torrent_review_requests.go
git commit -m "feat(review): add preview mapping endpoint for anidb"
```

---

## Task 3: Add Reject Endpoint to Backend

**Files:**
- Modify: `internal/dash/api/torrent_review_requests.go`

- [ ] **Step 3.1: Add reject request type**

After the preview types, add:

```go
type RejectReviewRequest struct {
	Reason string `json:"reason"`
}
```

- [ ] **Step 3.2: Add reject handler function**

```go
func handleRejectTorrentReviewRequest(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ErrorBadRequest(r, "invalid id").Send(w, r)
		return
	}

	review, err := torrent_mapping_review.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if review == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if review.Status != torrent_mapping_review.ReviewStatusPending {
		ErrorBadRequest(r, "review is not pending").Send(w, r)
		return
	}

	// Delete existing mappings
	if review.Target == torrent_mapping_review.MappingTargetIMDB {
		if review.PrevId != "" {
			imdb_torrent.Delete([]imdb_torrent.IMDBTorrent{{TId: review.PrevId, Hash: review.Hash}})
		}
	} else {
		if review.PrevId != "" {
			anidb.DeleteTorrentByTidAndHash(review.PrevId, review.Hash)
		}
	}

	if err := torrent_mapping_review.Reject(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, map[string]string{"status": "rejected"})
}
```

- [ ] **Step 3.3: Register reject endpoint**

In `AddTorrentReviewRequestsEndpoints` function, add:

```go
router.HandleFunc("/torrent/review-requests/{id}/reject", authed(handleRejectTorrentReviewRequest))
```

- [ ] **Step 3.4: Verify build**

Run: `go build ./internal/dash/api/...`
Expected: No errors

- [ ] **Step 3.5: Commit**

```bash
git add internal/dash/api/torrent_review_requests.go
git commit -m "feat(review): add reject endpoint for reviews"
```

---

## Task 4: Modify Resolve Endpoint for AniDB Mappings Array

**Files:**
- Modify: `internal/dash/api/torrent_review_requests.go`

- [ ] **Step 4.1: Update resolve payload type**

Replace `ResolveTorrentReviewPayload` with:

```go
type ResolveTorrentReviewPayload struct {
	MappingId string              `json:"mapping_id,omitempty"`
	Mappings  []AniDBMappingInput `json:"mappings,omitempty"`
}

type AniDBMappingInput struct {
	TId     string `json:"tid"`
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}
```

- [ ] **Step 4.2: Update AniDB resolve logic**

In `handleResolveTorrentReviewRequest`, replace the AniDB handling block (the else branch after IMDB) with:

```go
} else {
	// AniDB target
	if review.PrevId != "" {
		anidb.DeleteTorrentByTidAndHash(review.PrevId, review.Hash)
	}

	var items []anidb.AniDBTorrent

	if len(payload.Mappings) > 0 {
		// Use provided mappings
		for _, m := range payload.Mappings {
			eps := []int{}
			for i := m.EpStart; i <= m.EpEnd; i++ {
				eps = append(eps, i)
			}
			items = append(items, anidb.AniDBTorrent{
				TId:          m.TId,
				Hash:         review.Hash,
				SeasonType:   anidb.TorrentSeasonType(m.SType),
				Season:       m.S,
				EpisodeStart: m.EpStart,
				EpisodeEnd:   m.EpEnd,
				Episodes:     eps,
			})
		}
	} else if mappingId != "" {
		// Legacy: auto-map with forced ID
		tInfo, err := torrent_info.GetByHash(review.Hash)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if tInfo == nil {
			ErrorNotFound(r, "torrent info not found").Send(w, r)
			return
		}

		items = worker.MapTorrentToAniDB(review.Hash, *tInfo, func(msg string, e error, args ...any) {
			if e != nil {
				log.Error(msg, append([]any{"error", e}, args...)...)
			} else {
				log.Debug(msg, args...)
			}
		})

		for i := range items {
			items[i].TId = mappingId
		}

		if len(items) == 0 {
			items = []anidb.AniDBTorrent{{Hash: review.Hash, TId: mappingId}}
		}
	}

	if len(items) > 0 {
		if err := anidb.UpsertTorrents(items); err != nil {
			SendError(w, r, err)
			return
		}
	}
}
```

- [ ] **Step 4.3: Verify build**

Run: `go build ./internal/dash/api/...`
Expected: No errors

- [ ] **Step 4.4: Commit**

```bash
git add internal/dash/api/torrent_review_requests.go
git commit -m "feat(review): support mappings array in anidb resolve"
```

---

## Task 5: Add Preview and Reject API Calls to Frontend

**Files:**
- Modify: `apps/dash/src/api/torrent-review-requests.ts`

- [ ] **Step 5.1: Add preview types**

Add after existing types:

```typescript
export type PreviewMappingRequest = {
  anidb_id: string;
};

export type AniDBMappingPreview = {
  tid: string;
  s_type: "abs" | "tv" | "ani";
  s: number;
  ep_start: number;
  ep_end: number;
};

export type PreviewMappingResponse = {
  mappings: AniDBMappingPreview[];
  anidb_titles: string[];
};

export type RejectReviewRequest = {
  reason: string;
};
```

- [ ] **Step 5.2: Update resolve params type**

Replace `ResolveReviewRequestParams`:

```typescript
export type ResolveReviewRequestParams = {
  id: number;
  mapping_id?: string;
  mappings?: AniDBMappingPreview[];
};
```

- [ ] **Step 5.3: Add preview API function**

```typescript
async function previewMapping(
  id: number,
  params: PreviewMappingRequest,
): Promise<PreviewMappingResponse> {
  const { data } = await api<PreviewMappingResponse>(
    `/torrent/review-requests/${id}/preview`,
    {
      method: "POST",
      body: params,
    },
  );
  return data;
}
```

- [ ] **Step 5.4: Add reject API function**

```typescript
async function rejectReviewRequest(
  id: number,
  params: RejectReviewRequest,
): Promise<void> {
  await api(`/torrent/review-requests/${id}/reject`, {
    method: "POST",
    body: params,
  });
}
```

- [ ] **Step 5.5: Add usePreviewMapping hook**

```typescript
export function usePreviewMapping() {
  return useMutation({
    mutationFn: ({ id, params }: { id: number; params: PreviewMappingRequest }) =>
      previewMapping(id, params),
  });
}
```

- [ ] **Step 5.6: Add useRejectReviewRequest hook**

```typescript
export function useRejectReviewRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, params }: { id: number; params: RejectReviewRequest }) =>
      rejectReviewRequest(id, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/torrent/review-requests"] });
    },
  });
}
```

- [ ] **Step 5.7: Update resolveReviewRequest function**

Update the function body to handle mappings:

```typescript
async function resolveReviewRequest(params: ResolveReviewRequestParams) {
  const { id, ...body } = params;
  await api(`/torrent/review-requests/${id}/resolve`, {
    method: "PATCH",
    body,
  });
}
```

- [ ] **Step 5.8: Verify typecheck**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 5.9: Commit**

```bash
git add apps/dash/src/api/torrent-review-requests.ts
git commit -m "feat(review): add preview and reject api hooks"
```

---

## Task 6: Add Preview UI to Detail Sheet

**Files:**
- Modify: `apps/dash/src/components/torrent-review-request-detail-sheet.tsx`

- [ ] **Step 6.1: Add imports and types**

Update imports at top:

```typescript
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
```

- [ ] **Step 6.2: Add preview state**

After existing state declarations, add:

```typescript
const [previewMappings, setPreviewMappings] = useState<AniDBMappingPreview[] | null>(null);
const [isPreviewMode, setIsPreviewMode] = useState(false);

const previewMutation = usePreviewMapping();
const rejectMutation = useRejectReviewRequest();
```

- [ ] **Step 6.3: Add preview handler**

Add after `handleAniDBSelect`:

```typescript
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
```

- [ ] **Step 6.4: Add reject handler**

```typescript
async function handleReject() {
  if (!request) return;

  try {
    await rejectMutation.mutateAsync({
      id: request.id,
      params: { reason: "unmappable" },
    });
    toast.success("Review request rejected!");
    handleOpenChange(false);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to reject review.";
    toast.error(message);
  }
}
```

- [ ] **Step 6.5: Add mapping edit handlers**

```typescript
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
```

- [ ] **Step 6.6: Update handleResolve for mappings**

Replace `handleResolve` function:

```typescript
async function handleResolve() {
  if (!request) return;

  try {
    if (request.target === "anidb" && previewMappings) {
      // AniDB with preview mappings
      await resolveReviewRequest.mutateAsync({
        id: request.id,
        mappings: previewMappings,
      });
    } else {
      // IMDB or legacy flow
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
```

- [ ] **Step 6.7: Update handleOpenChange to reset preview state**

```typescript
function handleOpenChange(value: boolean) {
  if (!value) {
    setFinalMappingId("");
    setFinalMappingTitle("");
    setPreviewMappings(null);
    setIsPreviewMode(false);
  }
  onOpenChange(value);
}
```

- [ ] **Step 6.8: Commit partial progress**

```bash
git add apps/dash/src/components/torrent-review-request-detail-sheet.tsx
git commit -m "feat(review): add preview state and handlers to detail sheet"
```

---

## Task 7: Add Preview Table UI to Detail Sheet

**Files:**
- Modify: `apps/dash/src/components/torrent-review-request-detail-sheet.tsx`

- [ ] **Step 7.1: Create MappingPreviewTable component**

Add before the main component export:

```typescript
function MappingPreviewTable({
  mappings,
  onChange,
  onAdd,
  onRemove,
}: {
  mappings: AniDBMappingPreview[];
  onChange: (index: number, field: keyof AniDBMappingPreview, value: string | number) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-24">Type</TableHead>
            <TableHead className="w-16">S</TableHead>
            <TableHead className="w-20">Ep Start</TableHead>
            <TableHead className="w-20">Ep End</TableHead>
            <TableHead className="w-16"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {mappings.map((mapping, index) => (
            <TableRow key={index}>
              <TableCell>
                <Select
                  value={mapping.s_type}
                  onValueChange={(v) => onChange(index, "s_type", v)}
                >
                  <SelectTrigger className="h-8">
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
                  className="h-8 w-14"
                  value={mapping.s}
                  onChange={(e) => onChange(index, "s", parseInt(e.target.value) || 0)}
                />
              </TableCell>
              <TableCell>
                <Input
                  type="number"
                  className="h-8 w-16"
                  value={mapping.ep_start}
                  onChange={(e) => onChange(index, "ep_start", parseInt(e.target.value) || 0)}
                />
              </TableCell>
              <TableCell>
                <Input
                  type="number"
                  className="h-8 w-16"
                  value={mapping.ep_end}
                  onChange={(e) => onChange(index, "ep_end", parseInt(e.target.value) || 0)}
                />
              </TableCell>
              <TableCell>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemove(index)}
                >
                  ✕
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Button variant="outline" size="sm" onClick={onAdd}>
        + Add Row
      </Button>
    </div>
  );
}
```

- [ ] **Step 7.2: Update render for AniDB preview mode**

Replace the `request.status === "pending"` conditional block with:

```typescript
{request.status === "pending" && (
  <>
    {!isPreviewMode ? (
      <div className="flex flex-col gap-1.5">
        <Label>Final Mapping ID</Label>
        {request.target === "imdb" ? (
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
        {finalMappingId && (
          <p className="text-muted-foreground text-xs">{finalMappingId}</p>
        )}
        {request.target === "anidb" && finalMappingId && (
          <Button
            variant="outline"
            size="sm"
            disabled={previewMutation.isPending}
            onClick={handlePreview}
          >
            {previewMutation.isPending ? "Loading..." : "Preview Mappings"}
          </Button>
        )}
      </div>
    ) : (
      <div className="flex flex-col gap-2">
        <Label>Mapping Preview for {finalMappingId}</Label>
        {previewMappings && previewMappings.length > 0 ? (
          <MappingPreviewTable
            mappings={previewMappings}
            onChange={handleMappingChange}
            onAdd={handleAddMapping}
            onRemove={handleRemoveMapping}
          />
        ) : (
          <div className="text-muted-foreground text-sm">
            No mappings generated. Add manually or reject.
          </div>
        )}
        {(!previewMappings || previewMappings.length === 0) && (
          <Button variant="outline" size="sm" onClick={handleAddMapping}>
            + Add Row
          </Button>
        )}
      </div>
    )}
  </>
)}
```

- [ ] **Step 7.3: Update SheetFooter buttons**

Replace the SheetFooter content:

```typescript
<SheetFooter>
  <Button
    disabled={resolveReviewRequest.isPending || rejectMutation.isPending}
    onClick={() => handleOpenChange(false)}
    variant="outline"
  >
    Cancel
  </Button>
  {request?.status === "pending" && (
    <>
      {isPreviewMode && (
        <Button
          variant="outline"
          onClick={handleCancelPreview}
        >
          Back
        </Button>
      )}
      <Button
        variant="destructive"
        disabled={rejectMutation.isPending}
        onClick={handleReject}
      >
        {rejectMutation.isPending ? "Rejecting..." : "Reject"}
      </Button>
      {(request.target === "imdb" || isPreviewMode) && (
        <Button
          disabled={resolveReviewRequest.isPending}
          onClick={handleResolve}
        >
          {resolveReviewRequest.isPending ? "Resolving..." : "Apply"}
        </Button>
      )}
    </>
  )}
</SheetFooter>
```

- [ ] **Step 7.4: Verify typecheck**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 7.5: Commit**

```bash
git add apps/dash/src/components/torrent-review-request-detail-sheet.tsx
git commit -m "feat(review): add editable mapping preview table ui"
```

---

## Task 8: Add Validation and Edge Case Handling

**Files:**
- Modify: `apps/dash/src/components/torrent-review-request-detail-sheet.tsx`

- [ ] **Step 8.1: Add validation before apply**

Update `handleResolve` to validate:

```typescript
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
```

- [ ] **Step 8.2: Add empty state confirmation for reject**

Update `handleReject`:

```typescript
async function handleReject() {
  if (!request) return;

  if (previewMappings && previewMappings.length > 0) {
    if (!window.confirm("This will delete all mappings and mark as unmappable. Continue?")) {
      return;
    }
  }

  try {
    await rejectMutation.mutateAsync({
      id: request.id,
      params: { reason: "unmappable" },
    });
    toast.success("Review request rejected!");
    handleOpenChange(false);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Failed to reject review.";
    toast.error(message);
  }
}
```

- [ ] **Step 8.3: Verify typecheck**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 8.4: Commit**

```bash
git add apps/dash/src/components/torrent-review-request-detail-sheet.tsx
git commit -m "feat(review): add validation and confirmation for review resolution"
```

---

## Task 9: Final Verification

- [ ] **Step 9.1: Build backend**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 9.2: Build frontend**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 9.3: Manual testing checklist**

Test the following scenarios:
1. Open an AniDB review request in pending status
2. Search and select a corrected anidb_id
3. Click "Preview Mappings" - verify table appears
4. Edit an episode range
5. Add a new row
6. Delete a row
7. Click "Apply" - verify mappings saved
8. Test "Reject" flow
9. Test IMDB flow unchanged (no preview button)
10. Test resolved review shows read-only

- [ ] **Step 9.4: Final commit**

```bash
git add -A
git commit -m "feat(review): complete anidb review resolution with preview flow"
```

---

## Verification Summary

| Scenario | Expected Result |
|----------|-----------------|
| AniDB review → Preview | Shows editable table with generated mappings |
| Edit mapping row | Values update in table |
| Add row | New row appears with defaults |
| Delete row | Row removed from table |
| Apply with valid mappings | Old mappings deleted, new inserted, status resolved |
| Apply with ep_start > ep_end | Validation error, not submitted |
| Reject | Mappings deleted, status set to rejected |
| IMDB review | No preview button, direct resolve works |
| Already resolved review | All actions disabled |