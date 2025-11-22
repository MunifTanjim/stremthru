import { createFileRoute } from "@tanstack/react-router";

import { ListStatsCard } from "@/components/lists-stats-card";

export const Route = createFileRoute("/dash/lists/")({
  component: RouteComponent,
  staticData: {
    crumb: "Stats",
  },
});

function RouteComponent() {
  return (
    <>
      <ListStatsCard />
    </>
  );
}
