import { createFileRoute, Navigate, Outlet } from "@tanstack/react-router";

import { useServerStats } from "@/api/stats";

export const Route = createFileRoute("/dash/vault")({
  component: RouteComponent,
  staticData: {
    crumb: "Vault",
  },
});

function RouteComponent() {
  const { data: server } = useServerStats();

  if (server && !server.feature.vault) {
    return <Navigate to="/dash" />;
  }

  return <Outlet />;
}
