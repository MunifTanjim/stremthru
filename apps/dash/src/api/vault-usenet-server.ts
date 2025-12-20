import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type UsenetServer = {
  created_at: string;
  host: string;
  id: string;
  is_backup: boolean;
  max_connections: number;
  name: string;
  port: number;
  priority: number;
  tls: boolean;
  tls_skip_verify: boolean;
  updated_at: string;
  username: string;
};

type CreateUsenetServerParams = {
  host: string;
  is_backup: boolean;
  max_connections: number;
  name: string;
  password: string;
  port: number;
  priority: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
};

type PingUsenetServerParams = {
  host: string;
  id?: string;
  password: string;
  port: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
};

type PingUsenetServerResponse = {
  message: string;
  success: boolean;
};

type UpdateUsenetServerParams = Partial<{
  host: string;
  is_backup: boolean;
  max_connections: number;
  name: string;
  password: string;
  port: number;
  priority: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
}>;

export function useUsenetServerMutation() {
  const create = useMutation({
    mutationFn: createUsenetServer,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/usenet/servers"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      id,
      ...params
    }: UpdateUsenetServerParams & { id: string }) => {
      return updateUsenetServer(id, params);
    },
    onSuccess: async (data, _, __, ctx) => {
      ctx.client.setQueryData<UsenetServer[]>(
        ["/vault/usenet/servers"],
        (list) => list?.map((item) => (item.id === data.id ? data : item)),
      );
    },
  });

  const remove = useMutation({
    mutationFn: deleteUsenetServer,
    onSuccess: async (_, id, __, ctx) => {
      ctx.client.setQueryData<UsenetServer[]>(
        ["/vault/usenet/servers"],
        (list) => list?.filter((item) => item.id !== id),
      );
    },
  });

  const ping = useMutation({
    mutationFn: pingUsenetServer,
  });

  return { create, ping, remove, update };
}

export function useUsenetServers() {
  return useQuery({
    queryFn: getUsenetServers,
    queryKey: ["/vault/usenet/servers"],
  });
}

async function createUsenetServer(params: CreateUsenetServerParams) {
  const { data } = await api<UsenetServer>("POST /vault/usenet/servers", {
    body: params,
  });
  return data;
}

async function deleteUsenetServer(id: string) {
  await api(`DELETE /vault/usenet/servers/${id}`);
}

async function getUsenetServers() {
  const { data } = await api<UsenetServer[]>("/vault/usenet/servers");
  return data;
}

async function pingUsenetServer(params: PingUsenetServerParams) {
  const { data } = await api<PingUsenetServerResponse>(
    "POST /vault/usenet/servers/ping",
    { body: params },
  );
  return data;
}

async function updateUsenetServer(
  id: string,
  params: UpdateUsenetServerParams,
) {
  const { data } = await api<UsenetServer>(
    `PATCH /vault/usenet/servers/${id}`,
    { body: params },
  );
  return data;
}
