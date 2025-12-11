import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type CreateStremioTraktLinkParams = {
  stremio_account_id: string;
  sync_config: SyncConfig;
  trakt_account_id: string;
};

export type StremioTraktLink = {
  created_at: string;
  stremio_account_id: string;
  sync_config: SyncConfig;
  sync_state: SyncState;
  trakt_account_id: string;
  updated_at: string;
};

export type SyncConfig = {
  watched: SyncConfigWatched;
};

export type SyncConfigWatched = {
  dir: SyncDirection;
};

export type SyncDirection =
  | "both"
  | "none"
  | "stremio_to_trakt"
  | "trakt_to_stremio";

export type SyncState = {
  watched: SyncStateWatched;
};

export type SyncStateWatched = {
  last_synced_at?: string;
};

export type UpdateStremioTraktLinkParams = {
  sync_config: SyncConfig;
};

export function useStremioTraktLinkMutation() {
  const create = useMutation({
    mutationFn: createStremioTraktLink,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-trakt/links"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      stremio_account_id,
      trakt_account_id,
      ...params
    }: UpdateStremioTraktLinkParams & {
      stremio_account_id: string;
      trakt_account_id: string;
    }) => {
      return updateStremioTraktLink(
        stremio_account_id,
        trakt_account_id,
        params,
      );
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-trakt/links"],
      });
    },
  });

  const remove = useMutation({
    mutationFn: ({
      stremio_account_id,
      trakt_account_id,
    }: {
      stremio_account_id: string;
      trakt_account_id: string;
    }) => deleteStremioTraktLink(stremio_account_id, trakt_account_id),
    onSuccess: async (_, { stremio_account_id, trakt_account_id }, __, ctx) => {
      ctx.client.setQueryData<StremioTraktLink[]>(
        ["/sync/stremio-trakt/links"],
        (list) =>
          list?.filter(
            (item) =>
              item.stremio_account_id !== stremio_account_id ||
              item.trakt_account_id !== trakt_account_id,
          ),
      );
    },
  });

  const sync = useMutation({
    mutationFn: ({
      stremio_account_id,
      trakt_account_id,
    }: {
      stremio_account_id: string;
      trakt_account_id: string;
    }) => syncStremioTraktLink(stremio_account_id, trakt_account_id),
  });

  const resetSyncState = useMutation({
    mutationFn: ({
      stremio_account_id,
      trakt_account_id,
    }: {
      stremio_account_id: string;
      trakt_account_id: string;
    }) => resetStremioTraktLinkSyncState(stremio_account_id, trakt_account_id),
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-trakt/links"],
      });
    },
  });

  return { create, remove, resetSyncState, sync, update };
}

export function useStremioTraktLinks() {
  return useQuery({
    queryFn: getStremioTraktLinks,
    queryKey: ["/sync/stremio-trakt/links"],
  });
}

async function createStremioTraktLink(params: CreateStremioTraktLinkParams) {
  const { data } = await api<StremioTraktLink>(
    "POST /sync/stremio-trakt/links",
    {
      body: params,
    },
  );
  return data;
}

async function deleteStremioTraktLink(
  stremioAccountId: string,
  traktAccountId: string,
) {
  await api(
    `DELETE /sync/stremio-trakt/links/${stremioAccountId}:${traktAccountId}`,
  );
}

async function getStremioTraktLinks() {
  const { data } = await api<StremioTraktLink[]>("/sync/stremio-trakt/links");
  return data;
}

async function resetStremioTraktLinkSyncState(
  stremioAccountId: string,
  traktAccountId: string,
) {
  const { data } = await api<StremioTraktLink>(
    `POST /sync/stremio-trakt/links/${stremioAccountId}:${traktAccountId}/reset-sync-state`,
  );
  return data;
}

async function syncStremioTraktLink(
  stremioAccountId: string,
  traktAccountId: string,
) {
  await api(
    `POST /sync/stremio-trakt/links/${stremioAccountId}:${traktAccountId}/sync`,
  );
}

async function updateStremioTraktLink(
  stremioAccountId: string,
  traktAccountId: string,
  params: UpdateStremioTraktLinkParams,
) {
  const { data } = await api<StremioTraktLink>(
    `PATCH /sync/stremio-trakt/links/${stremioAccountId}:${traktAccountId}`,
    { body: params },
  );
  return data;
}
