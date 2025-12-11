import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type CreateStremioStremioLinkParams = {
  account_a_id: string;
  account_b_id: string;
  sync_config: SyncConfig;
};

export type StremioStremioLink = {
  account_a_id: string;
  account_b_id: string;
  created_at: string;
  sync_config: SyncConfig;
  sync_state: SyncState;
  updated_at: string;
};

export type SyncConfig = {
  watched: SyncConfigWatched;
};

export type SyncConfigWatched = {
  dir: SyncDirection;
  ids: string[];
};

export type SyncDirection = "a_to_b" | "b_to_a" | "both" | "none";

export type SyncState = {
  watched: SyncStateWatched;
};

export type SyncStateWatched = {
  last_synced_at?: string;
};

export type UpdateStremioStremioLinkParams = {
  sync_config: SyncConfig;
};

export function useStremioStremioLinkMutation() {
  const create = useMutation({
    mutationFn: createStremioStremioLink,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-stremio/links"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      account_a_id,
      account_b_id,
      ...params
    }: UpdateStremioStremioLinkParams & {
      account_a_id: string;
      account_b_id: string;
    }) => {
      return updateStremioStremioLink(account_a_id, account_b_id, params);
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-stremio/links"],
      });
    },
  });

  const remove = useMutation({
    mutationFn: ({
      account_a_id,
      account_b_id,
    }: {
      account_a_id: string;
      account_b_id: string;
    }) => deleteStremioStremioLink(account_a_id, account_b_id),
    onSuccess: async (_, { account_a_id, account_b_id }, __, ctx) => {
      ctx.client.setQueryData<StremioStremioLink[]>(
        ["/sync/stremio-stremio/links"],
        (list) =>
          list?.filter(
            (item) =>
              item.account_a_id !== account_a_id ||
              item.account_b_id !== account_b_id,
          ),
      );
    },
  });

  const sync = useMutation({
    mutationFn: ({
      account_a_id,
      account_b_id,
    }: {
      account_a_id: string;
      account_b_id: string;
    }) => syncStremioStremioLink(account_a_id, account_b_id),
  });

  const resetSyncState = useMutation({
    mutationFn: ({
      account_a_id,
      account_b_id,
    }: {
      account_a_id: string;
      account_b_id: string;
    }) => resetStremioStremioLinkSyncState(account_a_id, account_b_id),
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/sync/stremio-stremio/links"],
      });
    },
  });

  return { create, remove, resetSyncState, sync, update };
}

export function useStremioStremioLinks() {
  return useQuery({
    queryFn: getStremioStremioLinks,
    queryKey: ["/sync/stremio-stremio/links"],
  });
}

async function createStremioStremioLink(
  params: CreateStremioStremioLinkParams,
) {
  const { data } = await api<StremioStremioLink>(
    "POST /sync/stremio-stremio/links",
    {
      body: params,
    },
  );
  return data;
}

async function deleteStremioStremioLink(
  accountAId: string,
  accountBId: string,
) {
  await api(`DELETE /sync/stremio-stremio/links/${accountAId}:${accountBId}`);
}

async function getStremioStremioLinks() {
  const { data } = await api<StremioStremioLink[]>(
    "/sync/stremio-stremio/links",
  );
  return data;
}

async function resetStremioStremioLinkSyncState(
  accountAId: string,
  accountBId: string,
) {
  const { data } = await api<StremioStremioLink>(
    `POST /sync/stremio-stremio/links/${accountAId}:${accountBId}/reset-sync-state`,
  );
  return data;
}

async function syncStremioStremioLink(accountAId: string, accountBId: string) {
  await api(
    `POST /sync/stremio-stremio/links/${accountAId}:${accountBId}/sync`,
  );
}

async function updateStremioStremioLink(
  accountAId: string,
  accountBId: string,
  params: UpdateStremioStremioLinkParams,
) {
  const { data } = await api<StremioStremioLink>(
    `PATCH /sync/stremio-stremio/links/${accountAId}:${accountBId}`,
    { body: params },
  );
  return data;
}
