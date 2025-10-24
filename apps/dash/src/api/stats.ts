import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

type ServerStats = {
  started_at: string;
  version: string;
};

type TorrentsStats = {
  files: {
    total_count: number;
  };
  total_count: number;
};

export function useServerStats() {
  return useQuery({
    queryFn: getServerStats,
    queryKey: ["/stats/server"],
    staleTime: 2 * 60 * 60 * 1000,
  });
}

export function useTorrentsStats() {
  return useQuery({
    queryFn: getTorrentsStats,
    queryKey: ["/stats/torrents"],
    retry: false,
    staleTime: 2 * 60 * 60 * 1000,
  });
}

async function getServerStats() {
  const { data } = await api<ServerStats>("/stats/server");
  return data;
}

async function getTorrentsStats() {
  const { data } = await api<TorrentsStats>("/stats/torrents");
  return data;
}
