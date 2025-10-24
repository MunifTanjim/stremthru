import { createFileRoute } from "@tanstack/react-router";
import { Duration } from "luxon";
import { useMemo, useState } from "react";
import { useInterval } from "react-use";
import { Pie, PieChart } from "recharts";

import { useListsStats, useServerStats, useTorrentsStats } from "@/api/stats";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";

export const Route = createFileRoute("/dash/")({
  component: RouteComponent,
});

const listsChartConfig = {
  anilist: {
    color: "var(--chart-1)",
    label: "AniList",
  },
  items: {
    label: "Items",
  },
  letterboxd: {
    color: "var(--chart-3)",
    label: "Letterboxd",
  },
  lists: {
    label: "Lists",
  },
  mdblist: {
    color: "var(--chart-2)",
    label: "MDBList",
  },
  tmdb: {
    color: "var(--chart-4)",
    label: "TMDB",
  },
  trakt: {
    color: "var(--chart-1)",
    label: "Trakt",
  },
  tvdb: {
    color: "var(--chart-5)",
    label: "TVDB",
  },
} satisfies ChartConfig;

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

  const listsStats = useListsStats();
  const [listsCountChartData, listItemsCountChartData] = useMemo(() => {
    const data = listsStats.data;
    return [
      [
        {
          fill: "var(--color-anilist)",
          lists: data?.anilist.total_lists,
          service: "anilist",
        },
        {
          fill: "var(--color-mdblist)",
          lists: data?.mdblist.total_lists,
          service: "mdblist",
        },
        {
          fill: "var(--color-letterboxd)",
          lists: data?.letterboxd.total_lists,
          service: "letterboxd",
        },
        {
          fill: "var(--color-tmdb)",
          lists: data?.tmdb.total_lists,
          service: "tmdb",
        },
        {
          fill: "var(--color-trakt)",
          lists: data?.trakt.total_lists,
          service: "trakt",
        },
        {
          fill: "var(--color-tvdb)",
          lists: data?.tvdb.total_lists,
          service: "tvdb",
        },
      ],
      [
        {
          fill: "var(--color-anilist)",
          items: data?.anilist.total_items,
          service: "anilist",
        },
        {
          fill: "var(--color-mdblist)",
          items: data?.mdblist.total_items,
          service: "mdblist",
        },
        {
          fill: "var(--color-letterboxd)",
          items: data?.letterboxd.total_items,
          service: "letterboxd",
        },
        {
          fill: "var(--color-tmdb)",
          items: data?.tmdb.total_items,
          service: "tmdb",
        },
        {
          fill: "var(--color-trakt)",
          items: data?.trakt.total_items,
          service: "trakt",
        },
        {
          fill: "var(--color-tvdb)",
          items: data?.tvdb.total_items,
          service: "tvdb",
        },
      ],
    ];
  }, [listsStats.data]);

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

      <Card className="flex flex-col">
        <CardHeader className="items-center pb-0">
          <CardTitle>Lists Statistics</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 pb-0">
          <ChartContainer
            className="mx-auto aspect-square max-h-[250px]"
            config={listsChartConfig}
          >
            <PieChart>
              <ChartTooltip
                content={
                  <ChartTooltipContent indicator="dot" labelKey="service" />
                }
              />
              <Pie
                data={listsCountChartData}
                dataKey="lists"
                innerRadius={10}
                isAnimationActive
                name="Lists"
                outerRadius={50}
              />
              <Pie
                data={listItemsCountChartData}
                dataKey="items"
                innerRadius={60}
                isAnimationActive
                name="Items"
                outerRadius={90}
              />
            </PieChart>
          </ChartContainer>
        </CardContent>
        <CardFooter className="flex-col gap-2 text-sm">
          <div className="flex w-full grow flex-wrap">
            {listsCountChartData.map((ld, i) => (
              <div
                className="flex flex-1 grow flex-col justify-center gap-1 border px-6 py-4 text-left sm:px-8 sm:py-6"
                key={ld.service}
              >
                <span className="text-muted-foreground text-xs font-bold">
                  {listsChartConfig[ld.service].label}
                </span>
                <span className="whitespace-nowrap leading-none">
                  <span className="text-lg font-bold">
                    {ld.lists?.toLocaleString()}
                  </span>{" "}
                  Lists
                  <br />
                  <span className="text-nowrap text-lg font-bold">
                    {listItemsCountChartData[i].items?.toLocaleString()}
                  </span>{" "}
                  Items
                </span>
              </div>
            ))}
          </div>
        </CardFooter>
      </Card>
    </>
  );
}
