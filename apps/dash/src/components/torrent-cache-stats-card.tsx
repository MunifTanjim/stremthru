import { useTorrentsStats } from "@/api/stats";
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function TorrentCacheStatsCard() {
  const torzStats = useTorrentsStats();

  return (
    <Card className="py-4 2xl:py-0">
      <CardHeader className="flex flex-col items-stretch border-b !p-0 2xl:flex-row">
        <div className="flex flex-1 flex-col justify-center gap-1 px-6 pb-3 2xl:pb-0">
          <CardTitle>Cache Stats</CardTitle>
          <CardDescription>(since server start)</CardDescription>
        </div>
        <div className="flex flex-wrap">
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Torrent Info Write
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                (torzStats.data?.cache.write_torrent_info.miss.toLocaleString() ??
                0)
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Torrent Info Write Skip
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                <div className="flex items-end justify-between gap-1">
                  <span>
                    {torzStats.data?.cache.write_torrent_info.hit.toLocaleString() ??
                      0}
                  </span>{" "}
                  <span className="text-sm">
                    (
                    {(
                      100 *
                      ((torzStats.data?.cache.write_torrent_info.hit || 1) /
                        (torzStats.data?.cache.write_torrent_info.miss || 1))
                    ).toFixed(2)}
                    %)
                  </span>
                </div>
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Torrent Stream Write
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                (torzStats.data?.cache.write_torrent_stream.miss.toLocaleString() ??
                0)
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Torrent Stream Write Skip
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                <div className="flex items-end justify-between gap-1">
                  <span>
                    {torzStats.data?.cache.write_torrent_stream.hit.toLocaleString() ??
                      0}
                  </span>{" "}
                  <span className="text-sm">
                    (
                    {(
                      100 *
                      ((torzStats.data?.cache.write_torrent_stream.hit || 1) /
                        (torzStats.data?.cache.write_torrent_stream.miss || 1))
                    ).toFixed(2)}
                    %)
                  </span>
                </div>
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Magnet Cache Write
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                (torzStats.data?.cache.write_magnet_cache.miss.toLocaleString() ??
                0)
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Magnet Cache Write Skip
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                <div className="flex items-end justify-between gap-1">
                  <span>
                    {torzStats.data?.cache.write_magnet_cache.hit.toLocaleString() ??
                      0}
                  </span>{" "}
                  <span className="text-sm">
                    (
                    {(
                      100 *
                      ((torzStats.data?.cache.write_magnet_cache.hit || 1) /
                        (torzStats.data?.cache.write_magnet_cache.miss || 1))
                    ).toFixed(2)}
                    %)
                  </span>
                </div>
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Read Magnet Cache Hit
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                (torzStats.data?.cache.read_magnet_cache.hit.toLocaleString() ??
                0)
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Read Magnet Cache Miss
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                <div className="flex items-end justify-between gap-1">
                  <span>
                    {torzStats.data?.cache.read_magnet_cache.miss.toLocaleString() ??
                      0}
                  </span>
                </div>
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Read Torrent Stream Hit
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                (torzStats.data?.cache.read_torrent_stream.hit.toLocaleString() ??
                0)
              )}
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left 2xl:border-l 2xl:border-t-0 2xl:px-8 2xl:py-6">
            <span className="text-muted-foreground text-nowrap text-xs">
              Read Torrent Stream Miss
            </span>
            <span className="text-lg font-bold leading-none 2xl:text-3xl">
              {torzStats.isLoading ? (
                <Skeleton className="h-8 w-24" />
              ) : (
                <div className="flex items-end justify-between gap-1">
                  <span>
                    {torzStats.data?.cache.read_torrent_stream.miss.toLocaleString() ??
                      0}
                  </span>
                </div>
              )}
            </span>
          </div>
        </div>
      </CardHeader>
    </Card>
  );
}
