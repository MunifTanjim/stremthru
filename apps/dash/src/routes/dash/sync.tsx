import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/sync")({
  component: RouteComponent,
  staticData: {
    crumb: "Sync",
  },
});

function RouteComponent() {
  return <Outlet />;
}
