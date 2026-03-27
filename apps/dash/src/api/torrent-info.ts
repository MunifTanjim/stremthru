import { useInfiniteQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type TorrentInfoItem = {
  category: string;
  created_at: string;
  hash: string;
  imdb_id: string;
  indexer: string;
  leechers: number;
  private: boolean;
  seeders: number;
  size: number;
  src: string;
  t_title: string;
};

export type TorrentInfoListResponse = {
  items: TorrentInfoItem[];
  next_cursor: string;
};

export type TorrentInfoParams = {
  limit?: number;
  q?: string;
};

export function useTorrentInfos(params: TorrentInfoParams = {}) {
  const { limit = 20, q } = params;

  return useInfiniteQuery<
    TorrentInfoListResponse,
    Error,
    { pages: TorrentInfoListResponse[] },
    unknown[],
    string
  >({
    enabled: !!q,
    queryKey: ["/torrents/infos", { limit, q }],
    queryFn: ({ pageParam }) =>
      getTorrentInfos({ cursor: pageParam, limit, q }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });
}

async function getTorrentInfos(params: {
  cursor?: string;
  limit?: number;
  q?: string;
}) {
  const searchParams = new URLSearchParams();

  if (params.limit !== undefined) {
    searchParams.set("limit", params.limit.toString());
  }
  if (params.q) {
    searchParams.set("q", params.q);
  }
  if (params.cursor) {
    searchParams.set("cursor", params.cursor);
  }

  const query = searchParams.toString();
  const endpoint = `/torrents/infos${query ? `?${query}` : ""}` as const;
  const { data } = await api<TorrentInfoListResponse>(endpoint);
  return data;
}
