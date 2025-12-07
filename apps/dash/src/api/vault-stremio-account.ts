import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type CreateStremioAccountParams = {
  email: string;
  password: string;
};

export type StremioAccount = {
  created_at: string;
  email: string;
  id: string;
  is_valid: boolean;
  updated_at: string;
};

export type StremioAccountUserdata = {
  addon: "list" | "store" | "torz" | "wrap";
  created_at: string;
  key: string;
  name: string;
};

export type UpdateStremioAccountParams = {
  password: string;
};

export function useStremioAccountMutation() {
  const create = useMutation({
    mutationFn: createStremioAccount,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/stremio/accounts"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      id,
      ...params
    }: UpdateStremioAccountParams & { id: string }) => {
      return updateStremioAccount(id, params);
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/stremio/accounts"],
      });
    },
  });

  const remove = useMutation({
    mutationFn: deleteStremioAccount,
    onSuccess: async (_, id, __, ctx) => {
      const list = ctx.client.getQueryData<StremioAccount[]>([
        "/vault/stremio/accounts",
      ]);
      if (list) {
        ctx.client.setQueryData(
          ["/vault/stremio/accounts"],
          list.filter((item) => item.id !== id),
        );
      }
    },
  });

  const get = useMutation({
    mutationFn: getStremioAccount,
    onSuccess: async (data, { id }, __, ctx) => {
      ctx.client.setQueryData<StremioAccount[]>(
        ["/vault/stremio/accounts"],
        (list) =>
          list?.map((item) => (item.id === id ? { ...item, ...data } : item)),
      );
    },
  });

  const syncUserdata = useMutation({
    mutationFn: syncStremioAccountUserdata,
    onSuccess: async (items, id, __, ctx) => {
      ctx.client.setQueryData<StremioAccountUserdata[]>(
        ["/vault/stremio/accounts/{id}/userdata", id],
        items,
      );
    },
  });

  return { create, get, remove, syncUserdata, update };
}

export function useStremioAccounts() {
  return useQuery({
    queryFn: getStremioAccounts,
    queryKey: ["/vault/stremio/accounts"],
  });
}

export function useStremioAccountUserdata(id: string) {
  return useQuery({
    enabled: Boolean(id),
    queryFn: () => getStremioAccountUserdata(id),
    queryKey: ["/vault/stremio/accounts/{id}/userdata", id],
  });
}

async function createStremioAccount(params: CreateStremioAccountParams) {
  const { data } = await api<StremioAccount>("POST /vault/stremio/accounts", {
    body: params,
  });
  return data;
}

async function deleteStremioAccount(id: string) {
  await api(`DELETE /vault/stremio/accounts/${id}`);
}

async function getStremioAccount({
  id,
  refresh = false,
}: {
  id: string;
  refresh?: boolean;
}) {
  const { data } = await api<StremioAccount>(
    `GET /vault/stremio/accounts/${id}?refresh=${refresh}`,
  );
  return data;
}

async function getStremioAccounts() {
  const { data } = await api<StremioAccount[]>("/vault/stremio/accounts");
  return data;
}

async function getStremioAccountUserdata(id: string) {
  const { data } = await api<StremioAccountUserdata[]>(
    `GET /vault/stremio/accounts/${id}/userdata`,
  );
  return data;
}

async function syncStremioAccountUserdata(id: string) {
  const { data } = await api<StremioAccountUserdata[]>(
    `POST /vault/stremio/accounts/${id}/userdata/sync`,
  );
  return data;
}

async function updateStremioAccount(
  id: string,
  params: UpdateStremioAccountParams,
) {
  const { data } = await api<StremioAccount>(
    `PATCH /vault/stremio/accounts/${id}`,
    { body: params },
  );
  return data;
}
