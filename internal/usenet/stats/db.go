package usenet_stats

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/rs/xid"
)

const TableName = "usenet_server_stats"

var log = logger.Scoped("usenet/stats")

type UsenetServerStats struct {
	Id               string
	ServerId         string
	NZBHash          string
	SegmentsFetched  int64
	BytesDownloaded  int64
	ArticleNotFound  int64
	ConnectionErrors int64
	LatencySamples   []float64
	WallClockMs      float64
	CAt              db.Timestamp
}

var Column = struct {
	Id               string
	ServerId         string
	NZBHash          string
	SegmentsFetched  string
	BytesDownloaded  string
	ArticleNotFound  string
	ConnectionErrors string
	LatencySamples   string
	WallClockMs      string
	CAt              string
}{
	Id:               "id",
	ServerId:         "server_id",
	NZBHash:          "nzb_hash",
	SegmentsFetched:  "segments_fetched",
	BytesDownloaded:  "bytes_downloaded",
	ArticleNotFound:  "article_not_found",
	ConnectionErrors: "connection_errors",
	LatencySamples:   "latency_samples",
	WallClockMs:      "wall_clock_ms",
	CAt:              "cat",
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?,?,?,?,?,?,?)`,
	TableName,
	db.JoinColumnNames(
		Column.Id,
		Column.ServerId,
		Column.NZBHash,
		Column.SegmentsFetched,
		Column.BytesDownloaded,
		Column.ArticleNotFound,
		Column.ConnectionErrors,
		Column.LatencySamples,
		Column.WallClockMs,
	),
)

func Insert(stat *UsenetServerStats) error {
	if stat.Id == "" {
		stat.Id = xid.New().String()
	}
	_, err := db.Exec(query_insert,
		stat.Id, stat.ServerId, stat.NZBHash,
		stat.SegmentsFetched, stat.BytesDownloaded, stat.ArticleNotFound, stat.ConnectionErrors,
		db.JSONB[[]float64]{Data: stat.LatencySamples}, stat.WallClockMs,
	)
	return err
}

type AggregatedServerStats struct {
	ServerId         string  `json:"server_id"`
	ServerName       string  `json:"server_name"`
	SegmentsFetched  int64   `json:"segments_fetched"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	ArticleNotFound  int64   `json:"article_not_found"`
	ConnectionErrors int64   `json:"connection_errors"`
	ErrorRate        float64 `json:"error_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	ThroughputBps    float64 `json:"throughput_bps"`
	NZBCount         int64   `json:"nzb_count"`
	MissingNZBCount  int64   `json:"missing_nzb_count"`
}

func computeThroughput(wallClockBytes int64, totalWallClockMs float64, avgLatencyMs float64, segmentsFetched int64, bytesDownloaded int64) float64 {
	if totalWallClockMs > 0 {
		return float64(wallClockBytes) / (totalWallClockMs / 1000)
	}
	// fallback for old records without wall-clock data
	if avgLatencyMs > 0 && segmentsFetched > 0 {
		totalFetchTimeSec := avgLatencyMs * float64(segmentsFetched) / 1000
		return float64(bytesDownloaded) / totalFetchTimeSec
	}
	return 0
}

// percentile returns the p-th percentile from a sorted slice using linear interpolation.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	rank := (p / 100) * float64(n-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= n {
		return sorted[n-1]
	}
	frac := rank - float64(lower)
	return sorted[lower] + frac*(sorted[upper]-sorted[lower])
}

var query_get_aggregated_stats = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s >= ?`,
	db.JoinColumnNames(
		Column.ServerId,
		Column.NZBHash,
		Column.SegmentsFetched,
		Column.BytesDownloaded,
		Column.ArticleNotFound,
		Column.ConnectionErrors,
		Column.LatencySamples,
		Column.WallClockMs,
	),
	TableName,
	Column.CAt,
)

func GetAggregatedStats(since time.Time) ([]AggregatedServerStats, error) {
	rows, err := db.Query(query_get_aggregated_stats, db.Timestamp{Time: since})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type accumulator struct {
		stats            AggregatedServerStats
		latencySamples   []float64
		nzbHashes        map[string]struct{}
		missingNZBs      map[string]struct{}
		totalWallClockMs float64
		wallClockBytes   int64
	}
	accByServer := make(map[string]*accumulator)
	var serverOrder []string

	for rows.Next() {
		var serverId, nzbHash string
		var latencySamples db.JSONB[[]float64]
		var segFetched, bytesDown, artNotFound, connErrors int64
		var wallClockMs float64
		if err := rows.Scan(
			&serverId,
			&nzbHash,
			&segFetched,
			&bytesDown,
			&artNotFound,
			&connErrors,
			&latencySamples,
			&wallClockMs,
		); err != nil {
			return nil, err
		}

		acc, ok := accByServer[serverId]
		if !ok {
			acc = &accumulator{
				stats:       AggregatedServerStats{ServerId: serverId},
				nzbHashes:   make(map[string]struct{}),
				missingNZBs: make(map[string]struct{}),
			}
			accByServer[serverId] = acc
			serverOrder = append(serverOrder, serverId)
		}

		acc.stats.SegmentsFetched += segFetched
		acc.stats.BytesDownloaded += bytesDown
		acc.stats.ArticleNotFound += artNotFound
		acc.stats.ConnectionErrors += connErrors

		if wallClockMs > 0 {
			acc.totalWallClockMs += wallClockMs
			acc.wallClockBytes += bytesDown
		}

		if len(latencySamples.Data) > 0 {
			acc.latencySamples = append(acc.latencySamples, latencySamples.Data...)
		}

		acc.nzbHashes[nzbHash] = struct{}{}
		if artNotFound > 0 {
			acc.missingNZBs[nzbHash] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	slices.Sort(serverOrder)

	result := make([]AggregatedServerStats, 0, len(accByServer))
	for _, id := range serverOrder {
		acc := accByServer[id]
		s := &acc.stats
		if len(acc.latencySamples) > 0 {
			slices.Sort(acc.latencySamples)
			var total float64
			for _, v := range acc.latencySamples {
				total += v
			}
			s.AvgLatencyMs = total / float64(len(acc.latencySamples))
			s.P50LatencyMs = percentile(acc.latencySamples, 50)
			s.P95LatencyMs = percentile(acc.latencySamples, 95)
			s.P99LatencyMs = percentile(acc.latencySamples, 99)
		}
		s.ThroughputBps = computeThroughput(acc.wallClockBytes, acc.totalWallClockMs, s.AvgLatencyMs, s.SegmentsFetched, s.BytesDownloaded)
		s.NZBCount = int64(len(acc.nzbHashes))
		s.MissingNZBCount = int64(len(acc.missingNZBs))
		totalOps := s.SegmentsFetched + s.ArticleNotFound + s.ConnectionErrors
		if totalOps > 0 {
			s.ErrorRate = math.Round(float64(s.ArticleNotFound+s.ConnectionErrors)/float64(totalOps)*10000) / 100
		}
		result = append(result, *s)
	}
	return result, nil
}

type TimeSeriesBucket struct {
	Time             string  `json:"time"`
	SegmentsFetched  int64   `json:"segments_fetched"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	ArticleNotFound  int64   `json:"article_not_found"`
	ConnectionErrors int64   `json:"connection_errors"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ThroughputBps    float64 `json:"throughput_bps"`
}

type ServerTimeSeries struct {
	Name    string             `json:"name"`
	Buckets []TimeSeriesBucket `json:"buckets"`
}

var query_get_time_series_stats = fmt.Sprintf(
	`SELECT %s, %%s AS bucket, %s FROM %s WHERE %s >= ? ORDER BY %s, bucket`,
	Column.ServerId,
	db.JoinColumnNames(
		Column.SegmentsFetched,
		Column.BytesDownloaded,
		Column.ArticleNotFound,
		Column.ConnectionErrors,
		Column.LatencySamples,
		Column.WallClockMs,
	),
	TableName,
	Column.CAt,
	Column.ServerId,
)

func GetTimeSeriesStats(since time.Time, interval string) (map[string]*ServerTimeSeries, error) {
	var bucketExpr string
	switch db.Dialect {
	case db.DBDialectSQLite:
		switch interval {
		case "hour":
			bucketExpr = `strftime('%Y-%m-%dT%H:00:00Z', cat, 'unixepoch')`
		default:
			bucketExpr = `strftime('%Y-%m-%dT00:00:00Z', cat, 'unixepoch')`
		}
	case db.DBDialectPostgres:
		switch interval {
		case "hour":
			bucketExpr = `to_char(date_trunc('hour', cat), 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
		default:
			bucketExpr = `to_char(date_trunc('day', cat), 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
		}
	default:
		return nil, fmt.Errorf("unsupported db dialect: %s", db.Dialect)
	}

	query := fmt.Sprintf(query_get_time_series_stats, bucketExpr)

	rows, err := db.Query(query, db.Timestamp{Time: since})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type bucketKey struct {
		ServerId string
		Bucket   string
	}
	type accumulator struct {
		bucket           TimeSeriesBucket
		latencySamples   []float64
		totalWallClockMs float64
		wallClockBytes   int64
	}

	result := make(map[string]*ServerTimeSeries)
	accByKey := make(map[bucketKey]*accumulator)
	var keyOrder []bucketKey

	for rows.Next() {
		var serverId, bucket, samplesJSON string
		var segFetched, bytesDown, artNotFound, connErrors int64
		var wallClockMs float64
		if err := rows.Scan(&serverId, &bucket, &segFetched, &bytesDown, &artNotFound, &connErrors, &samplesJSON, &wallClockMs); err != nil {
			return nil, err
		}

		if _, ok := result[serverId]; !ok {
			result[serverId] = &ServerTimeSeries{}
		}

		key := bucketKey{ServerId: serverId, Bucket: bucket}
		acc, ok := accByKey[key]
		if !ok {
			acc = &accumulator{bucket: TimeSeriesBucket{Time: bucket}}
			accByKey[key] = acc
			keyOrder = append(keyOrder, key)
		}

		acc.bucket.SegmentsFetched += segFetched
		acc.bucket.BytesDownloaded += bytesDown
		acc.bucket.ArticleNotFound += artNotFound
		acc.bucket.ConnectionErrors += connErrors

		if wallClockMs > 0 {
			acc.totalWallClockMs += wallClockMs
			acc.wallClockBytes += bytesDown
		}

		var samples []float64
		if samplesJSON != "" {
			if err := json.Unmarshal([]byte(samplesJSON), &samples); err == nil {
				acc.latencySamples = append(acc.latencySamples, samples...)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, key := range keyOrder {
		acc := accByKey[key]
		if len(acc.latencySamples) > 0 {
			var total float64
			for _, v := range acc.latencySamples {
				total += v
			}
			acc.bucket.AvgLatencyMs = math.Round(total/float64(len(acc.latencySamples))*100) / 100
		}
		acc.bucket.ThroughputBps = computeThroughput(acc.wallClockBytes, acc.totalWallClockMs, acc.bucket.AvgLatencyMs, acc.bucket.SegmentsFetched, acc.bucket.BytesDownloaded)
		result[key.ServerId].Buckets = append(result[key.ServerId].Buckets, acc.bucket)
	}

	return result, nil
}

var query_delete_older_than = fmt.Sprintf(
	`DELETE FROM %s WHERE %s < ?`,
	TableName,
	Column.CAt,
)

func DeleteOlderThan(t time.Time) (int64, error) {
	res, err := db.Exec(query_delete_older_than, db.Timestamp{Time: t})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func flushAccumulators() {
	drained := drainAllAccumulators()
	if len(drained) == 0 {
		return
	}

	globalMu.Lock()
	servers := make(map[string]ServerInfo, len(globalServers))
	for k, v := range globalServers {
		servers[k] = v
	}
	globalMu.Unlock()

	for key, acc := range drained {
		segFetched, bytesDown, missingSegments, connErrors, durations, wallClockMs := acc.drain()

		if segFetched == 0 && missingSegments == 0 && connErrors == 0 {
			continue
		}

		info := servers[key.ProviderId]

		stat := &UsenetServerStats{
			ServerId:         info.Id,
			NZBHash:          key.NZBHash,
			SegmentsFetched:  segFetched,
			BytesDownloaded:  bytesDown,
			ArticleNotFound:  missingSegments,
			ConnectionErrors: connErrors,
			LatencySamples:   durations,
			WallClockMs:      wallClockMs,
		}

		if err := Insert(stat); err != nil {
			log.Error("failed to flush usenet server stats", "error", err, "server_id", info.Id, "nzb_hash", key.NZBHash)
		}
	}
}

var backgroundJobQuit chan struct{}

func CleanupBackgroundJob() {
	if backgroundJobQuit != nil {
		close(backgroundJobQuit)
	}
	flushAccumulators()
}

func initBackgroundJob() {
	backgroundJobQuit = make(chan struct{})
	go func() {
		flushTicker := time.NewTicker(5 * time.Minute)
		cleanupTicker := time.NewTicker(2 * time.Minute)
		defer flushTicker.Stop()
		defer cleanupTicker.Stop()

		for {
			select {
			case <-backgroundJobQuit:
				return
			case <-flushTicker.C:
				flushAccumulators()
			case <-cleanupTicker.C:
				cutoff := time.Now().AddDate(0, 0, -90)
				if count, err := DeleteOlderThan(cutoff); err != nil {
					log.Error("failed to cleanup old usenet server stats", "error", err)
				} else if count > 0 {
					log.Info("cleaned up old usenet server stats", "count", count)
				}
			}
		}
	}()
}

var initBackgroundJobOnce sync.Once

func InitBackgroundJob() {
	initBackgroundJobOnce.Do(initBackgroundJob)
}
