import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

import { api } from "@/lib/api";

export type AniDBMappingItem = {
  anidb_id: string;
  anidb_title: string;
  ep_end: number;
  ep_start: number;
  hash: string;
  mapped_at: string;
  season: number;
  season_type: string;
  t_title: string;
};

export type IMDBMappingItem = {
  hash: string;
  imdb_id: string;
  imdb_title: string;
  imdb_type: string;
  imdb_year: number;
  mapped_at: string;
  t_title: string;
};

export type MappingMode = "by-id" | "by-title";

export type MappingParams = {
  limit?: number;
  mode: MappingMode;
  q: string;
  unmapped?: boolean;
};

export type ReprocessRequest = {
  hashes: string[];
  targets?: ReprocessTarget[];
};

export type ReprocessResponse = {
  mapped?: { anidb?: number; imdb?: number };
  mode: "async" | "sync";
  parsed?: number;
  processed?: number;
  queued?: number;
};

export type ReprocessTarget = "anidb" | "imdb";

type MappingsListResponse<T> = {
  items: T[];
  next_cursor: string;
};

export function useAniDBMappings(params: MappingParams) {
  const { limit = 100, mode, q, unmapped } = params;

  return useInfiniteQuery<
    MappingsListResponse<AniDBMappingItem>,
    Error,
    { pages: MappingsListResponse<AniDBMappingItem>[] },
    unknown[],
    string
  >({
    enabled: !!q,
    queryKey: ["/torrents/info/anidb", { limit, mode, q, unmapped }],
    queryFn: ({ pageParam }) =>
      getAniDBMappings({ cursor: pageParam, limit, mode, q, unmapped }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });
}

export function useIMDBMappings(params: MappingParams) {
  const { limit = 100, mode, q, unmapped } = params;

  return useInfiniteQuery<
    MappingsListResponse<IMDBMappingItem>,
    Error,
    { pages: MappingsListResponse<IMDBMappingItem>[] },
    unknown[],
    string
  >({
    enabled: !!q,
    queryKey: ["/torrents/info/imdb", { limit, mode, q, unmapped }],
    queryFn: ({ pageParam }) =>
      getIMDBMappings({ cursor: pageParam, limit, mode, q, unmapped }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });
}

export function useReprocessTorrents() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: reprocessTorrents,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/torrents/info/imdb"] });
      queryClient.invalidateQueries({ queryKey: ["/torrents/info/anidb"] });
    },
  });
}

async function getAniDBMappings(params: {
  cursor?: string;
  limit?: number;
  mode: MappingMode;
  q: string;
  unmapped?: boolean;
}) {
  const searchParams = new URLSearchParams();
  searchParams.set("mode", params.mode);
  searchParams.set("q", params.q);
  if (params.limit !== undefined) {
    searchParams.set("limit", params.limit.toString());
  }
  if (params.cursor) {
    searchParams.set("cursor", params.cursor);
  }
  if (params.unmapped !== undefined) {
    searchParams.set("unmapped", params.unmapped.toString());
  }

  const query = searchParams.toString();
  const endpoint = `/torrents/info/anidb${query ? `?${query}` : ""}` as const;
  const { data } = await api<MappingsListResponse<AniDBMappingItem>>(endpoint);
  return data;
}

async function getIMDBMappings(params: {
  cursor?: string;
  limit?: number;
  mode: MappingMode;
  q: string;
  unmapped?: boolean;
}) {
  const searchParams = new URLSearchParams();
  searchParams.set("mode", params.mode);
  searchParams.set("q", params.q);
  if (params.limit !== undefined) {
    searchParams.set("limit", params.limit.toString());
  }
  if (params.cursor) {
    searchParams.set("cursor", params.cursor);
  }
  if (params.unmapped !== undefined) {
    searchParams.set("unmapped", params.unmapped.toString());
  }

  const query = searchParams.toString();
  const endpoint = `/torrents/info/imdb${query ? `?${query}` : ""}` as const;
  const { data } = await api<MappingsListResponse<IMDBMappingItem>>(endpoint);
  return data;
}

async function reprocessTorrents(params: ReprocessRequest) {
  const { data } = await api<ReprocessResponse>("/torrents/reprocess", {
    body: params,
    method: "POST",
  });
  return data;
}
