import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type WorkerDetails = Record<
  string,
  {
    has_failed_job: boolean;
    id: string;
    interval: number;
    title: string;
  }
>;

export type WorkerJobLog = {
  created_at: string;
  data?: unknown;
  error?: string;
  id: string;
  name: string;
  status: "done" | "failed" | "started";
  updated_at: string;
};

export type WorkerTemporaryFile = {
  modified_at: string;
  path: string;
  size: string;
};

export function useWorkerDetails() {
  return useQuery({
    queryFn: getWorkerDetails,
    queryKey: ["/workers/details"],
  });
}

export function useWorkerJobLogs(workerId: string) {
  return useQuery({
    enabled: Boolean(workerId),
    queryFn: () => getWorkerJobLogs(workerId),
    queryKey: ["/workers/{id}/job-logs", workerId],
  });
}

export function useWorkerMutation(workerId: string) {
  const queryClient = useQueryClient();

  const purgeJobLogs = useMutation({
    mutationFn: async () => {
      await api(`/workers/${workerId}/job-logs`, { method: "DELETE" });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["/workers/{id}/job-logs", workerId],
      });
    },
  });

  const purgeTemporaryFiles = useMutation({
    mutationFn: async () => {
      await api(`/workers/${workerId}/temporary-files`, { method: "DELETE" });
    },
  });

  return { purgeJobLogs, purgeTemporaryFiles };
}

export function useWorkerTemporaryFiles(workerId: string) {
  return useQuery({
    enabled: Boolean(workerId),
    queryFn: async () => {
      const { data } = await api<WorkerTemporaryFile[]>(
        `/workers/${workerId}/temporary-files`,
      );
      return data;
    },
    queryKey: ["/workers/{id}/temporary-files", workerId],
  });
}

async function getWorkerDetails() {
  const { data } = await api<WorkerDetails>("/workers/details");
  return data;
}

async function getWorkerJobLogs(workerId: string) {
  const { data } = await api<WorkerJobLog[]>(`/workers/${workerId}/job-logs`);
  return data;
}
