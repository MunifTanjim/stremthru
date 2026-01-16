import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type NzbQueueItem = {
  category: string;
  created_at: string;
  error: string;
  id: string;
  name: string;
  priority: number;
  status: string;
  updated_at: string;
  url: string;
  user: string;
};

export function useNzbQueue() {
  return useQuery({
    queryFn: getNzbQueueItems,
    queryKey: ["/usenet/queue"],
    refetchInterval: 10 * 60 * 1000,
  });
}

export function useNzbQueueMutation() {
  const remove = useMutation({
    mutationFn: deleteNzbQueueItem,
    onSuccess: async (_, id, __, ctx) => {
      ctx.client.setQueryData<NzbQueueItem[]>(["/usenet/queue"], (list) =>
        list?.filter((item) => item.id !== id),
      );
    },
  });

  return { remove };
}

async function deleteNzbQueueItem(id: string) {
  await api(`DELETE /usenet/queue/${id}`);
}

async function getNzbQueueItems() {
  const { data } = await api<NzbQueueItem[]>("/usenet/queue");
  return data;
}
