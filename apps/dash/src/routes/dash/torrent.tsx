import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/torrent")({
  component: RouteComponent,
  staticData: {
    crumb: "Torrent",
  },
});

function RouteComponent() {
  return <Outlet />;
}
