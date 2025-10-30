import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type WorkerDetails = Record<
  string,
  {
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

async function getWorkerDetails() {
  const { data } = await api<WorkerDetails>("/workers/details");
  return data;
}

async function getWorkerJobLogs(workerId: string) {
  const { data } = await api<WorkerJobLog[]>(`/workers/${workerId}/job-logs`);
  return data;
}
