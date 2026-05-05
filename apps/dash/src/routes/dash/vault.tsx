import { createFileRoute, Navigate, Outlet } from "@tanstack/react-router";

import { useFeature } from "@/hooks/use-feature";

export const Route = createFileRoute("/dash/vault")({
  component: RouteComponent,
  staticData: {
    crumb: "Vault",
  },
});

function RouteComponent() {
  const features = useFeature();

  if (features.isReady && !features.get("vault")) {
    return <Navigate to="/dash" />;
  }

  return <Outlet />;
}
