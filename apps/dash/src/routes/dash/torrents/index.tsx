import { createFileRoute } from "@tanstack/react-router";

import { TorrentCacheStatsCard } from "@/components/torrent-cache-stats-card";
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

      <TorrentCacheStatsCard />
    </>
  );
}
