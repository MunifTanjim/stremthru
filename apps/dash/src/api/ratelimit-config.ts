import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type RateLimitConfig = {
  created_at: string;
  id: string;
  limit: number;
  name: string;
  updated_at: string;
  window: string;
};

export function useRateLimitConfig(id: null | string): null | RateLimitConfig {
  const { data } = useRateLimitConfigs();
  return data?.find((item) => item.id === id) ?? null;
}

export function useRateLimitConfigMutation() {
  const create = useMutation({
    mutationFn: createRateLimitConfig,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/ratelimit/configs"],
      });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      id,
      ...params
    }: Pick<RateLimitConfig, "limit" | "name" | "window"> & { id: string }) => {
      return updateRateLimitConfig(id, params);
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/ratelimit/configs"],
      });
    },
  });

  const remove = useMutation({
    mutationFn: deleteRateLimitConfig,
    onSuccess: async (_, id, __, ctx) => {
      ctx.client.setQueryData<RateLimitConfig[]>(
        ["/ratelimit/configs"],
        (list) => list?.filter((item) => item.id !== id),
      );
    },
  });

  return { create, remove, update };
}

export function useRateLimitConfigs() {
  return useQuery({
    queryFn: getRateLimitConfigs,
    queryKey: ["/ratelimit/configs"],
  });
}

async function createRateLimitConfig(
  params: Pick<RateLimitConfig, "limit" | "name" | "window">,
) {
  const { data } = await api<RateLimitConfig>("POST /ratelimit/configs", {
    body: params,
  });
  return data;
}

async function deleteRateLimitConfig(id: string) {
  await api(`DELETE /ratelimit/configs/${id}`);
}

async function getRateLimitConfigs() {
  const { data } = await api<RateLimitConfig[]>("/ratelimit/configs");
  return data;
}

async function updateRateLimitConfig(
  id: string,
  params: Pick<RateLimitConfig, "limit" | "name" | "window">,
) {
  const { data } = await api<RateLimitConfig>(
    `PATCH /ratelimit/configs/${id}`,
    { body: params },
  );
  return data;
}
