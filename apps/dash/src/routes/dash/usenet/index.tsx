import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/usenet/")({
  component: RouteComponent,
});

function RouteComponent() {
  return null;
}
