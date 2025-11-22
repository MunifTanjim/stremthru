import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/torrents")({
  component: RouteComponent,
  staticData: {
    crumb: "Torrents",
  },
});

function RouteComponent() {
  return <Outlet />;
}
