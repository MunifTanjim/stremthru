import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

const MINUTE = 60 * 1000;

export type AggregatedServerStats = {
  article_not_found: number;
  avg_latency_ms: number;
  bytes_downloaded: number;
  connection_errors: number;
  error_rate: number;
  missing_nzb_count: number;
  nzb_count: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  segments_fetched: number;
  server_id: string;
  server_name: string;
  throughput_bps: number;
};

export type ServerTimeSeries = {
  buckets: TimeSeriesBucket[];
  name: string;
};

export type TimeSeriesBucket = {
  article_not_found: number;
  avg_latency_ms: number;
  bytes_downloaded: number;
  connection_errors: number;
  segments_fetched: number;
  throughput_bps: number;
  time: string;
};

type UsenetServerStatsHistoryData = {
  items: AggregatedServerStats[];
};

type UsenetServerStatsTimeSeriesData = {
  items: Record<string, ServerTimeSeries>;
};

export function useUsenetServerStatsHistory(range: string) {
  return useQuery({
    queryFn: async () => {
      const { data } = await api<UsenetServerStatsHistoryData>(
        `/stats/usenet-servers/history?range=${range}`,
      );
      return data;
    },
    queryKey: ["/stats/usenet-servers/history", range],
    staleTime: 5 * MINUTE,
  });
}

export function useUsenetServerStatsTimeSeries(range: string) {
  return useQuery({
    queryFn: async () => {
      const { data } = await api<UsenetServerStatsTimeSeriesData>(
        `/stats/usenet-servers/timeseries?range=${range}`,
      );
      return data;
    },
    queryKey: ["/stats/usenet-servers/timeseries", range],
    staleTime: 5 * MINUTE,
  });
}
