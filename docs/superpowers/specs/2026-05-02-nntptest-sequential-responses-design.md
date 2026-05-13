# Sequential Mock Responses for nntptest.Server

## Purpose

Enable mock NNTP server to return different responses for the same command on subsequent calls. Supports testing retry logic, round-robin behavior, and failure-then-success scenarios.

## Current State

- `responses map[string]response` stores single response per command
- `SetResponse(cmd, statusLine, body...)` overwrites any existing response
- `getResponse(cmd)` returns the single configured response

## Design

### Data Structure Changes

```go
type Server struct {
    // ...existing fields...
    responses      map[string][]response  // slice of responses per command
    responseIndex  map[string]int         // tracks consumption position per command
    // ...
}
```

### Behavior

**SetResponse:**

- Appends response to `responses[command]` slice
- Signature unchanged: `SetResponse(command, statusLine string, body ...[]string)`
- Multiple calls for same command queue responses in order

**getResponse:**

- Returns `responses[cmd][index]` where index = `responseIndex[cmd]`
- Advances index after each call
- Index capped at `len(responses[cmd]) - 1` (repeats last response forever after exhaustion)

**Initialization:**

- `NewServer` initializes both maps empty
- `responseIndex` entries created lazily (default 0)

### Example Usage

```go
server := nntptest.NewServer(t, "200 Ready")

// First call returns error, subsequent calls succeed
server.SetResponse("BODY <msg@test>", "400 Service unavailable")
server.SetResponse("BODY <msg@test>", "222 0 <msg@test>", bodyLines)

// Call 1: returns 400
// Call 2+: returns 222 with body
```

## Files to Modify

- `internal/nntp/nntptest/server.go`

## Testing

Existing pool tests (`TestFetchSegment_*`) already exercise sequential response behavior and will validate the implementation.
