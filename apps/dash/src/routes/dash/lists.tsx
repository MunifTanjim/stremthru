import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/lists")({
  component: RouteComponent,
  staticData: {
    crumb: "Lists",
  },
});

function RouteComponent() {
  return <Outlet />;
}
