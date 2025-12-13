import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type TorznabIndexer = {
  created_at: string;
  id: string;
  name: string;
  type: TorznabIndexerType;
  updated_at: string;
  url: string;
};

export type TorznabIndexerType = "jackett";

type CreateTorznabIndexerParams = {
  api_key: string;
  name: string;
  type?: TorznabIndexerType;
  url: string;
};

type UpdateTorznabIndexerParams = {
  api_key?: string;
  name?: string;
};

export function useTorznabIndexerMutation() {
  const create = useMutation({
    mutationFn: createTorznabIndexer,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/torznab/indexers"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      id,
      ...params
    }: UpdateTorznabIndexerParams & { id: string }) => {
      return updateTorznabIndexer(id, params);
    },
    onSuccess: async (data, __, ___, ctx) => {
      ctx.client.setQueryData<TorznabIndexer[]>(
        ["/vault/torznab/indexers"],
        (items) => items?.map((item) => (item.id == data.id ? data : item)),
      );
    },
  });

  const remove = useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      return deleteTorznabIndexer(id);
    },
    onSuccess: async (_, { id }, __, ctx) => {
      ctx.client.setQueryData<TorznabIndexer[]>(
        ["/vault/torznab/indexers"],
        (list) => list?.filter((item) => item.id !== id),
      );
    },
  });

  const test = useMutation({
    mutationFn: testTorznabIndexer,
  });

  return { create, remove, test, update };
}

export function useTorznabIndexers() {
  return useQuery({
    queryFn: getTorznabIndexers,
    queryKey: ["/vault/torznab/indexers"],
  });
}

async function createTorznabIndexer(params: CreateTorznabIndexerParams) {
  const { data } = await api<TorznabIndexer>(`POST /vault/torznab/indexers`, {
    body: params,
  });
  return data;
}

async function deleteTorznabIndexer(id: string) {
  await api(`DELETE /vault/torznab/indexers/${id}`);
}

async function getTorznabIndexers() {
  const { data } = await api<TorznabIndexer[]>(`/vault/torznab/indexers`);
  return data;
}

async function testTorznabIndexer(id: string) {
  const { data } = await api<TorznabIndexer>(
    `POST /vault/torznab/indexers/${id}/test`,
  );
  return data;
}

async function updateTorznabIndexer(
  id: string,
  params: UpdateTorznabIndexerParams,
) {
  const { data } = await api<TorznabIndexer>(
    `PATCH /vault/torznab/indexers/${id}`,
    { body: params },
  );
  return data;
}
