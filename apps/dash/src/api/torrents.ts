import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type RequestTorrentReviewFile = {
  ep: number;
  path: string;
  prev_ep: number;
  prev_s: number;
  s: number;
};

export type RequestTorrentReviewItem = {
  comment?: string;
  files?: RequestTorrentReviewFile[];
  hash: string;
  imdb_id?: string;
  prev_imdb_id?: string;
  reason: RequestTorrentReviewReason;
};

export type RequestTorrentReviewPayload = {
  items: RequestTorrentReviewItem[];
};

export type RequestTorrentReviewReason =
  | "fake_torrent"
  | "incomplete_season_pack"
  | "other"
  | "wrong_mapping"
  | "wrong_title";

export type Torrent = {
  files?: TorrentFile[];
  hash: string;
  name: string;
  private: boolean;
  seeders: number;
  size: string;
};

export type TorrentFile = {
  aisd?: string;
  index: number;
  name: string;
  path: string;
  sid?: string;
  size: number;
};

export function useRequestTorrentReview() {
  return useMutation({
    mutationFn: async (payload: RequestTorrentReviewPayload) => {
      await api<null>("/torrents/review", {
        body: payload,
        method: "POST",
      });
    },
  });
}

export function useTorrents(imdbId?: string) {
  return useQuery({
    enabled: !!imdbId,
    queryFn: async () => {
      const { data } = await api<Torrent[]>(
        `/torrents?imdbid=${encodeURIComponent(imdbId!)}`,
      );
      return data;
    },
    queryKey: ["/torrents", imdbId],
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
