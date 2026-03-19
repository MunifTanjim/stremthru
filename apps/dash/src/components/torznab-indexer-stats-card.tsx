import { useTorznabIndexerStats } from "@/api/stats";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function TorznabIndexerStatsCard() {
  const stats = useTorznabIndexerStats();

  return (
    <Card className="py-4 sm:py-0">
      <CardHeader className="flex flex-col items-stretch border-b !p-0 sm:flex-row">
        <div className="flex flex-1 flex-col justify-center gap-1 px-6 pb-3 sm:pb-0">
          <CardTitle>Torznab Indexer Statistics</CardTitle>
          <CardDescription>Overview of indexer sync status</CardDescription>
        </div>
        <div className="flex flex-wrap">
          {(
            [
              ["Total", "total_count"],
              ["Synced", "synced_count"],
              ["Queued", "queued_count"],
              ["Errors", "error_count"],
              ["Results", "result_count"],
            ] as const
          ).map(([label, key]) => (
            <div
              className="flex flex-1 flex-col justify-center gap-1 border-l border-t px-6 py-4 text-left sm:border-l sm:border-t-0 sm:px-8 sm:py-6"
              key={key}
            >
              <span className="text-muted-foreground text-xs">{label}</span>
              <span className="text-lg font-bold leading-none sm:text-3xl">
                {stats.isLoading ? (
                  <Skeleton className="h-8 w-24" />
                ) : (
                  (stats.data?.[key]?.toLocaleString() ?? 0)
                )}
              </span>
            </div>
          ))}
        </div>
      </CardHeader>
      <CardContent className="px-2 pb-4">
        {stats.isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : stats.data?.indexers.length ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead className="text-right">Synced</TableHead>
                <TableHead className="text-right">Queued</TableHead>
                <TableHead className="text-right">Errors</TableHead>
                <TableHead className="text-right">Results</TableHead>
                <TableHead className="text-right">Avg. Results/Item</TableHead>
                <TableHead className="text-right">Last Synced</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {stats.data.indexers.map((indexer) => (
                <TableRow key={indexer.indexer_id}>
                  <TableCell className="font-medium">
                    {indexer.indexer_name}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.total_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.synced_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.queued_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.error_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.result_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {(indexer.result_count / indexer.synced_count).toFixed(2)}
                  </TableCell>
                  <TableCell className="text-right">
                    {indexer.last_synced_at
                      ? new Date(indexer.last_synced_at).toLocaleString()
                      : "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}
      </CardContent>
    </Card>
  );
}
