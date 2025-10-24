import { createFileRoute } from "@tanstack/react-router";
import { Duration } from "luxon";
import { useState } from "react";
import { useInterval } from "react-use";

import { useServerStats, useTorrentsStats } from "@/api/stats";
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export const Route = createFileRoute("/dash/")({
  component: RouteComponent,
});

function RouteComponent() {
  const torzStats = useTorrentsStats();
  const serverStats = useServerStats();

  const [uptime, setUptime] = useState("");
  useInterval(() => {
    if (!serverStats.data) {
      return;
    }
    const uptime = Duration.fromMillis(
      Date.now() - new Date(serverStats.data.started_at).getTime(),
    );
    setUptime(
      uptime
        .shiftTo("months", "days", "hours", "minutes", "seconds")
        .removeZeros()
        .toHuman({
          maximumFractionDigits: 0,
          unitDisplay: "short",
        }),
    );
  }, 1000);

  return (
    <>
      <Card>
        <CardHeader>
          <CardDescription>Server Uptime</CardDescription>
          <CardTitle className="@[250px]/card:text-3xl text-2xl font-semibold tabular-nums">
            {!uptime || serverStats.isLoading ? (
              <Skeleton className="h-8 w-48" />
            ) : (
              uptime
            )}
          </CardTitle>
        </CardHeader>
      </Card>
      <Card className="py-4 sm:py-0">
        <CardHeader className="flex flex-col items-stretch border-b !p-0 sm:flex-row">
          <div className="flex flex-1 flex-col justify-center gap-1 px-6 pb-3 sm:pb-0">
            <CardTitle>Torrents Statistics</CardTitle>
            <CardDescription>Overview of torrents in database</CardDescription>
          </div>
          <div className="flex">
            <div className="flex flex-1 flex-col justify-center gap-1 border-t px-6 py-4 text-left even:border-l sm:border-l sm:border-t-0 sm:px-8 sm:py-6">
              <span className="text-muted-foreground text-xs">
                Total Torrents
              </span>
              <span className="text-lg font-bold leading-none sm:text-3xl">
                {torzStats.isLoading ? (
                  <Skeleton className="h-8 w-24" />
                ) : (
                  (torzStats.data?.total_count.toLocaleString() ?? 0)
                )}
              </span>
            </div>
            <div className="flex flex-1 flex-col justify-center gap-1 border-t px-6 py-4 text-left even:border-l sm:border-l sm:border-t-0 sm:px-8 sm:py-6">
              <span className="text-muted-foreground text-xs">Total Files</span>
              <span className="text-lg font-bold leading-none sm:text-3xl">
                {torzStats.isLoading ? (
                  <Skeleton className="h-8 w-24" />
                ) : (
                  (torzStats.data?.files.total_count.toLocaleString() ?? 0)
                )}
              </span>
            </div>
          </div>
        </CardHeader>
      </Card>
    </>
  );
}
