import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/settings/")({
  component: RouteComponent,
});

function RouteComponent() {
  return null;
}
