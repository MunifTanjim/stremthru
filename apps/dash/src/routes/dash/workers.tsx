import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/dash/workers")({
  component: RouteComponent,
  staticData: {
    crumb: "Workers",
  },
});

function RouteComponent() {
  return null;
}
