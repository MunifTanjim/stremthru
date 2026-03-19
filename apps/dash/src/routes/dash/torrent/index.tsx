import { createFileRoute } from "@tanstack/react-router";

import { TorrentCacheStatsCard } from "@/components/torrent-cache-stats-card";
import { TorrentsStatsCard } from "@/components/torrents-stats-card";
import { TorznabIndexerStatsCard } from "@/components/torznab-indexer-stats-card";

export const Route = createFileRoute("/dash/torrent/")({
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

      <TorznabIndexerStatsCard />
    </>
  );
}
