import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/sync/")({
  component: RouteComponent,
});

function RouteComponent() {
  return null;
}
