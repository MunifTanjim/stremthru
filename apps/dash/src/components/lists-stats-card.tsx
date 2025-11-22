import { useMemo } from "react";
import { Pie, PieChart } from "recharts";

import { useListsStats } from "@/api/stats";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";

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
    color: "var(--chart-1)",
    label: "TMDB",
  },
  trakt: {
    color: "var(--chart-4)",
    label: "Trakt",
  },
  tvdb: {
    color: "var(--chart-5)",
    label: "TVDB",
  },
} satisfies ChartConfig;

export function ListStatsCard() {
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
    <Card className="flex flex-col">
      <CardHeader className="items-center pb-0">
        <CardTitle>Lists Statistics</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col flex-wrap pb-0 md:flex-row">
        <ChartContainer
          className="mx-auto aspect-square max-h-[250px] min-w-[250px] flex-1 shrink-0 grow"
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
        <div className="flex flex-1 flex-wrap">
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
      </CardContent>
    </Card>
  );
}
