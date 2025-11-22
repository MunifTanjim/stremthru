import { createFileRoute } from "@tanstack/react-router";

import { TorrentsStatsCard } from "@/components/torrents-stats-card";

export const Route = createFileRoute("/dash/torrents/")({
  component: RouteComponent,
  staticData: {
    crumb: "Stats",
  },
});

function RouteComponent() {
  return (
    <>
      <TorrentsStatsCard />
    </>
  );
}
