import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type TorznabIndexerSyncInfo = {
  error: null | string;
  id: string;
  queued_at: null | string;
  result_count: null | number;
  sid: string;
  synced_at: null | string;
  type: string;
};

export type TorznabIndexerSyncInfoListResponse = {
  items: TorznabIndexerSyncInfo[];
  total_count: number;
};

export type TorznabIndexerSyncInfoParams = {
  limit?: number;
  offset?: number;
  sid?: string;
};

export function useTorznabIndexerSyncInfoMutation() {
  const queue = useMutation({
    mutationFn: async (sid: string) => {
      await api(`POST /torrents/indexer-syncinfos`, {
        body: { sid },
      });
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/torrents/indexer-syncinfos"],
      });
    },
  });

  return { queue };
}

export function useTorznabIndexerSyncInfos(
  params: TorznabIndexerSyncInfoParams = {},
) {
  const { limit = 100, offset = 0, sid } = params;

  return useQuery({
    queryFn: () => getTorznabIndexerSyncInfos({ limit, offset, sid }),
    queryKey: ["/torrents/indexer-syncinfos", { limit, offset, sid }],
  });
}

async function getTorznabIndexerSyncInfos(
  params: TorznabIndexerSyncInfoParams,
) {
  const searchParams = new URLSearchParams();

  if (params.limit) {
    searchParams.set("limit", params.limit.toString());
  }
  if (params.offset) {
    searchParams.set("offset", params.offset.toString());
  }
  if (params.sid) {
    searchParams.set("sid", params.sid);
  }

  const query = searchParams.toString();
  const endpoint =
    `/torrents/indexer-syncinfos${query ? `?${query}` : ""}` as const;
  const { data } = await api<TorznabIndexerSyncInfoListResponse>(endpoint);
  return data;
}
