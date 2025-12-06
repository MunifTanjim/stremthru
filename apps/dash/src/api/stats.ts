import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

type IMDBTitleStats = {
  total_count: number;
};

type ListsStats = Record<
  "anilist" | "letterboxd" | "mdblist" | "tmdb" | "trakt" | "tvdb",
  {
    total_items: number;
    total_lists: number;
  }
>;

type ServerStats = {
  feature: {
    vault: boolean;
  };
  started_at: string;
  version: string;
};

type TorrentsStats = {
  files: {
    total_count: number;
  };
  total_count: number;
};

const HOUR = 60 * 60 * 1000;

export function useIMDBTitleStats() {
  return useQuery({
    queryFn: async () => {
      const { data } = await api<IMDBTitleStats>("/stats/imdb-titles");
      return data;
    },
    queryKey: ["/stats/imdb-titles"],
    staleTime: 2 * HOUR,
  });
}

export function useListsStats() {
  return useQuery({
    queryFn: getListsStats,
    queryKey: ["/stats/lists"],
    staleTime: 2 * HOUR,
  });
}

export function useServerStats() {
  return useQuery({
    queryFn: getServerStats,
    queryKey: ["/stats/server"],
    staleTime: 2 * HOUR,
  });
}

export function useTorrentsStats() {
  return useQuery({
    queryFn: getTorrentsStats,
    queryKey: ["/stats/torrents"],
    retry: false,
    staleTime: 2 * HOUR,
  });
}

async function getListsStats() {
  const { data } = await api<ListsStats>("/stats/lists");
  return data;
}

async function getServerStats() {
  const { data } = await api<ServerStats>("/stats/server");
  return data;
}

async function getTorrentsStats() {
  const { data } = await api<TorrentsStats>("/stats/torrents");
  return data;
}
