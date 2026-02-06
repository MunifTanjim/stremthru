import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type NewznabIndexer = {
  created_at: string;
  disabled: boolean;
  id: number;
  name: string;
  rate_limit_config_id: null | string;
  type: NewznabIndexerType;
  updated_at: string;
  url: string;
};

type CreateNewznabIndexerParams = {
  api_key: string;
  name: string;
  rate_limit_config_id: null | string;
  url: string;
};

type NewznabIndexerType = "generic";

type UpdateNewznabIndexerParams = {
  api_key?: string;
  name?: string;
  rate_limit_config_id: null | string;
};

export function useNewznabIndexerMutation() {
  const create = useMutation({
    mutationFn: createNewznabIndexer,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/newznab/indexers"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      id,
      ...params
    }: UpdateNewznabIndexerParams & { id: number }) => {
      return updateNewznabIndexer(id, params);
    },
    onSuccess: async (data, __, ___, ctx) => {
      ctx.client.setQueryData<NewznabIndexer[]>(
        ["/vault/newznab/indexers"],
        (items) => items?.map((item) => (item.id == data.id ? data : item)),
      );
    },
  });

  const remove = useMutation({
    mutationFn: async ({ id }: { id: number }) => {
      return deleteNewznabIndexer(id);
    },
    onSuccess: async (_, { id }, __, ctx) => {
      ctx.client.setQueryData<NewznabIndexer[]>(
        ["/vault/newznab/indexers"],
        (list) => list?.filter((item) => item.id !== id),
      );
    },
  });

  const test = useMutation({
    mutationFn: testNewznabIndexer,
  });

  const toggle = useMutation({
    mutationFn: toggleNewznabIndexer,
    onSuccess: async (data, _, __, ctx) => {
      ctx.client.setQueryData<NewznabIndexer[]>(
        ["/vault/newznab/indexers"],
        (items) => items?.map((item) => (item.id == data.id ? data : item)),
      );
    },
  });

  return { create, remove, test, toggle, update };
}

export function useNewznabIndexers() {
  return useQuery({
    queryFn: getNewznabIndexers,
    queryKey: ["/vault/newznab/indexers"],
  });
}

async function createNewznabIndexer(params: CreateNewznabIndexerParams) {
  const { data } = await api<NewznabIndexer>(`POST /vault/newznab/indexers`, {
    body: params,
  });
  return data;
}

async function deleteNewznabIndexer(id: number) {
  await api(`DELETE /vault/newznab/indexers/${id}`);
}

async function getNewznabIndexers() {
  const { data } = await api<NewznabIndexer[]>(`/vault/newznab/indexers`);
  return data;
}

async function testNewznabIndexer(id: number) {
  const { data } = await api<NewznabIndexer>(
    `POST /vault/newznab/indexers/${id}/test`,
  );
  return data;
}

async function toggleNewznabIndexer(id: number) {
  const { data } = await api<NewznabIndexer>(
    `POST /vault/newznab/indexers/${id}/toggle`,
  );
  return data;
}

async function updateNewznabIndexer(
  id: number,
  params: UpdateNewznabIndexerParams,
) {
  const { data } = await api<NewznabIndexer>(
    `PATCH /vault/newznab/indexers/${id}`,
    { body: params },
  );
  return data;
}
