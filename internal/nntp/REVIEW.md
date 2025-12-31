# Code Review: StreamFile Feature

**Date:** 2025-12-23
**Scope:** `internal/nntp/stream.go`, `usenet_pool.go`, `pool.go`

## Overview

The `StreamFile` feature implements a streaming file reader for Usenet/NNTP that:
- Streams NZB file segments from multiple Usenet providers
- Implements `io.ReadSeekCloser` for HTTP range request support
- Uses parallel prefetch workers for performance
- Has LRU segment caching
- Supports provider failover

**Overall Assessment**: The architecture is solid and composable. The separation of concerns (UsenetPool -> Pool -> Connection) is good. However, there are several issues that should be addressed.

---

## Critical Issues

### 1. Potential Duplicate Segment Fetch (stream.go:284-303)

```go
func (ufr *UsenetFileReader) getSegmentData(idx int) ([]byte, error) {
    if seg, ok := ufr.cache.get(idx); ok {
        <-seg.ready  // BLOCKS while holding no lock
        // ...
    }
    // Fetch synchronously if not in cache
    data, err := ufr.fetchSegment(ufr.ctx, ufr.segments[idx])
```

**Problem**: If segment isn't in cache, it fetches synchronously WITHOUT claiming it first. Meanwhile, a prefetch worker could also try to fetch the same segment, leading to:
- Duplicate network requests for the same segment
- Wasted bandwidth and connections

**Recommendation**: Use `hasOrClaiming` before synchronous fetch to prevent duplicate work.

---

### 2. Data Size Mismatch Not Validated (stream.go:270-277)

```go
data, err := decodeYEnc(article.Body)
conn.Release()
if err != nil {
    return nil, err
}
return data, nil  // No size validation!
```

**Problem**: The `nzb.Segment.Bytes` field indicates expected size, but decoded data isn't validated against it. This could cause:
- Silent data corruption going undetected
- Out-of-bounds reads in `Read()` at line 392: `segData[offsetInSeg:offsetInSeg+int64(bytesToCopy)]`

**Recommendation**: Validate decoded data size matches `seg.Bytes`.

---

### 3. Potential Panic on Out-of-Bounds Slice (stream.go:392)

```go
copy(p[totalRead:], segData[offsetInSeg:offsetInSeg+int64(bytesToCopy)])
```

**Problem**: If `segData` is smaller than expected (see issue #2), this will panic with index out of range.

**Recommendation**: Add bounds checking or handle size mismatch gracefully.

---

## Moderate Issues

### 4. Cache Stores Segment Twice (stream.go:296-302)

```go
// Fetch synchronously if not in cache
data, err := ufr.fetchSegment(ufr.ctx, ufr.segments[idx])
if err != nil {
    ufr.cache.store(idx, nil, err)  // Stores error
    return nil, err
}
ufr.cache.store(idx, data, nil)  // Stores success
```

**Problem**: `store()` adds to LRU every time (line 142), so if a segment is stored twice (once in sync path, once in prefetch), it appears twice in LRU list.

**Recommendation**: Either use `hasOrClaiming` to prevent double-fetch, or make `store()` idempotent for the LRU list.

---

### 5. Prefetch Channel Size May Be Suboptimal (stream.go:307)

```go
ufr.prefetchCh = make(chan int, ufr.parallelism*2)
```

**Problem**: With `parallelism=4` and `bufferAhead=2`, the channel has capacity 8, but only 4 segments are triggered per read cycle. This is fine, but the relationship between `bufferAhead` and channel size is unclear.

**Recommendation**: Consider `bufferAhead + parallelism` as channel size for clearer semantics.

---

### 6. LRU Implementation Is O(n) (stream.go:147-157)

```go
func (sc *segmentCache) touchLRU(idx int) {
    for i, v := range sc.lru {  // Linear scan
        if v == idx {
            sc.lru = append(sc.lru[:i], sc.lru[i+1:]...)
            break
        }
    }
    sc.lru = append(sc.lru, idx)
}
```

**Problem**: Linear search for every cache hit. With typical cache sizes (6-10 segments), this is acceptable, but doesn't scale.

**Recommendation**: For production use with larger caches, consider a proper LRU implementation with O(1) operations (linked list + map).

---

### 7. Groups Selection Only Uses First Group (stream.go:236-243)

```go
if len(ufr.groups) > 0 && conn.CurrentGroup() != ufr.groups[0] {
    if _, err := conn.Group(ufr.groups[0]); err != nil {
```

**Problem**: NZB files can list multiple groups where an article is posted. If the first group fails, other groups aren't tried.

**Recommendation**: Iterate through groups on group selection failure.

---

### 8. Connection Not Released Before yEnc Decode (stream.go:269-276)

```go
// Fetch article body
article, err := conn.Body(messageId)
if err != nil { ... }

// Decode yEnc
data, err := decodeYEnc(article.Body)
conn.Release()  // Released AFTER decode
```

**Problem**: Connection is held during yEnc decode, which is CPU-bound. This unnecessarily blocks other requests from using this connection.

**Recommendation**: Read the body into a buffer and release connection before decoding.

---

## Minor Issues / Improvements

### 9. No Context Timeout on Segment Fetch

`fetchSegment` uses the reader's context but doesn't add per-segment timeout. A slow/hung server could block a worker indefinitely.

**Recommendation**: Add per-segment fetch timeout (e.g., 30s).

---

### 10. Error Caching Could Lead to Retry Storm

When an error is cached, subsequent reads for that segment will immediately fail. However, the error might be transient.

**Recommendation**: Consider TTL on cached errors or retry logic.

---

### 11. No Metrics/Observability

No counters for cache hits/misses, segment fetch times, retry counts, or provider failovers.

**Recommendation**: Add structured logging or metrics for debugging production issues.

---

### 12. `triggerPrefetch` Uses Parallelism Instead of BufferAhead

```go
func (ufr *UsenetFileReader) triggerPrefetch(startIdx int) {
    for i := 0; i < ufr.parallelism && startIdx+i < len(ufr.segments); i++ {
```

**Problem**: Should probably use `bufferAhead` or `parallelism + bufferAhead` to prefetch more segments ahead of current read position.

---

## Composability Assessment

### Strengths

1. **Clean Interface**: Returns `io.ReadSeekCloser` - works with standard library and HTTP handlers
2. **Provider Abstraction**: `UsenetPool` cleanly abstracts multiple providers with failover
3. **Configurable**: Parallelism and buffer size are configurable
4. **Test Coverage**: Good unit tests for cache, seek, and offset calculations

### Areas for Improvement

1. **No Progress Callback**: Consumers can't observe download progress
2. **No Cancellation Feedback**: No way to know if Close() was due to cancellation vs completion
3. **Segment Interface**: Consider interface for segments to allow testing with mock segments

---

## Summary

| Priority | Issue | Impact |
|----------|-------|--------|
| Critical | Duplicate segment fetch possible | Wasted resources, race conditions |
| Critical | No size validation on decoded data | Silent corruption, potential panic |
| High | Connection held during decode | Reduced throughput |
| Medium | LRU has duplicate entries | Memory inefficiency |
| Medium | Only first group tried | Reduced availability |
| Low | O(n) LRU | Performance at scale |
| Low | No per-segment timeout | Potential hangs |