import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type MaintenanceStatus = {
  ends_at: string;
  is_active: boolean;
};

export function useMaintenanceMutation() {
  const queryClient = useQueryClient();

  const activate = useMutation({
    mutationFn: activateMaintenance,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["/maintenance"] });
    },
  });

  const deactivate = useMutation({
    mutationFn: deactivateMaintenance,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["/maintenance"] });
    },
  });

  return { activate, deactivate };
}

export function useMaintenanceStatus() {
  return useQuery({
    queryFn: getMaintenanceStatus,
    queryKey: ["/maintenance"],
    refetchInterval: 10_000,
    staleTime: 10_000,
  });
}

async function activateMaintenance(params?: { duration: string }) {
  const { data } = await api<MaintenanceStatus>("POST /maintenance", {
    body: params ?? {},
  });
  return data;
}

async function deactivateMaintenance() {
  const { data } = await api<MaintenanceStatus>("DELETE /maintenance");
  return data;
}

async function getMaintenanceStatus() {
  const { data } = await api<MaintenanceStatus>("/maintenance");
  return data;
}
