# Requeue All NZBs Button

## Context

Users need a way to requeue all NZBs at once from the Usenet NZB dashboard, rather than clicking requeue individually on each item. This is useful when NNTP provider configuration changes or when retrying after transient failures.

## Requirements

- Button in dashboard header (right side)
- Requeues all NZBs in database (not just visible/filtered)
- Excludes NZBs currently downloading
- Confirmation dialog before execution
- Shows count of requeued items on success

## Design

### Backend

**New endpoint:** `POST /usenet/nzb/requeue-all`

**Location:** `internal/dash/api/usenet-nzb.go`

**Handler logic:**

1. Fetch all NZBs where status ≠ "downloading" and URL exists
2. For each NZB, call `nzb_info.QueueJob()` with original user/name/URL/password
3. Return `{ "count": N }`

### Frontend

**File:** `apps/dash/src/routes/dash/usenet/nzb.tsx`

**Changes:**

1. Add mutation for `POST /usenet/nzb/requeue-all` in `apps/dash/src/api/nzb-info.ts`
2. Add button with `RefreshCw` icon + "Requeue All" text to header right side
3. Wrap button in `AlertDialog` for confirmation
4. Dialog text: "Requeue all NZBs?" / "This will re-process all NZBs (excluding those currently downloading)."
5. On success: toast "Requeued N NZBs", invalidate `/usenet/nzb` and `/usenet/queue` queries
6. Button disabled while mutation pending

## Verification

1. Start dev server
2. Navigate to Usenet NZB dashboard
3. Click "Requeue All" button
4. Verify confirmation dialog appears
5. Confirm and verify toast shows correct count
6. Verify NZB list refreshes with updated statuses
