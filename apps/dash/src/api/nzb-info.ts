import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type NZBContentFile = {
  alias?: string;
  errors?: string[];
  files?: NZBContentFile[];
  name: string;
  parts?: NZBContentFile[];
  size: number;
  streamable: boolean;
  type: string;
  volume?: number;
};

export type NZBInfoItem = {
  cached: boolean;
  created_at: string;
  date: string;
  file_count: number;
  files: null | NZBContentFile[];
  hash: string;
  id: string;
  name: string;
  password: string;
  size: number;
  status: string;
  streamable: boolean;
  updated_at: string;
  url: string;
  user: string;
};

export function useNzbInfo() {
  return useQuery({
    queryFn: getNzbInfoItems,
    queryKey: ["/usenet/nzb"],
    refetchInterval: 10 * 60 * 1000,
  });
}

export function useNzbInfoMutation() {
  const remove = useMutation({
    mutationFn: deleteNzbInfoItem,
    onSuccess: async (_, id, __, ctx) => {
      ctx.client.setQueryData<NZBInfoItem[]>(["/usenet/nzb"], (list) =>
        list?.filter((item) => item.id !== id),
      );
    },
  });

  const requeue = useMutation({
    mutationFn: requeueNzbInfoItem,
    onSuccess: async (_, _id, __, ctx) => {
      await ctx.client.invalidateQueries({ queryKey: ["/usenet/nzb"] });
      await ctx.client.invalidateQueries({ queryKey: ["/usenet/queue"] });
    },
  });

  const requeueAll = useMutation({
    mutationFn: requeueAllNzbInfoItems,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({ queryKey: ["/usenet/nzb"] });
      await ctx.client.invalidateQueries({ queryKey: ["/usenet/queue"] });
    },
  });

  return { remove, requeue, requeueAll };
}

async function deleteNzbInfoItem(id: string) {
  await api(`DELETE /usenet/nzb/${id}`);
}

async function getNzbInfoItems() {
  const { data } = await api<NZBInfoItem[]>("/usenet/nzb");
  return data;
}

async function requeueAllNzbInfoItems() {
  const { data } = await api<{ count: number }>("POST /usenet/nzb/requeue-all");
  return data;
}

async function requeueNzbInfoItem(id: string) {
  await api(`POST /usenet/nzb/${id}/requeue`);
}
