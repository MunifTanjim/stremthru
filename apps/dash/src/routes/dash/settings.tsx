import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/settings")({
  component: RouteComponent,
  staticData: {
    crumb: "Settings",
  },
});

function RouteComponent() {
  return <Outlet />;
}
