import { createFileRoute, Link } from "@tanstack/react-router";
import { HammerIcon, RefreshCwIcon } from "lucide-react";
import { toast } from "sonner";

import {
  type UsenetConfig,
  type UsenetPoolProviderInfo,
  useRebuildUsenetPoolMutation,
  useUsenetConfig,
  useUsenetPoolInfo,
} from "@/api/usenet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemTitle,
} from "@/components/ui/item";
import { Spinner } from "@/components/ui/spinner";
import { APIError } from "@/lib/api";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/dash/usenet/config")({
  component: RouteComponent,
  staticData: {
    crumb: "Config",
  },
});

type SimpleConfigKey = Exclude<keyof UsenetConfig, "indexer_request_header">;

const configFields: { key: SimpleConfigKey; label: string }[] = [
  { key: "nzb_cache_size", label: "NZB Cache Size" },
  { key: "nzb_cache_ttl", label: "NZB Cache TTL" },
  { key: "nzb_max_file_size", label: "NZB Max File Size" },
  { key: "segment_cache_size", label: "Segment Cache Size" },
  { key: "stream_buffer_size", label: "Stream Buffer Size" },
  { key: "max_connection_per_stream", label: "Max Connection Per Stream" },
];

const queryTypeLabels: Record<string, string> = {
  "*": "Any/Fallback",
  movie: "Movie",
  tv: "TV",
};

function getStateBadgeVariant(
  state: UsenetPoolProviderInfo["state"],
): "default" | "destructive" | "outline" | "secondary" {
  switch (state) {
    case "auth_failed":
    case "offline":
      return "destructive";
    case "connecting":
      return "secondary";
    case "disabled":
      return "outline";
    case "online":
      return "default";
  }
}

function getStateColor(state: UsenetPoolProviderInfo["state"]) {
  switch (state) {
    case "auth_failed":
      return "bg-red-500";
    case "connecting":
      return "bg-yellow-500";
    case "disabled":
      return "bg-gray-500";
    case "offline":
      return "bg-red-500";
    case "online":
      return "bg-green-500";
  }
}

function getStateLabel(state: UsenetPoolProviderInfo["state"]) {
  switch (state) {
    case "auth_failed":
      return "Auth Failed";
    case "connecting":
      return "Connecting";
    case "disabled":
      return "Disabled";
    case "offline":
      return "Offline";
    case "online":
      return "Online";
  }
}

function HeaderTable({ headers }: { headers: Record<string, string> }) {
  const entries = Object.entries(headers);
  if (entries.length === 0) {
    return <div className="text-muted-foreground text-sm">None</div>;
  }
  return (
    <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
      {entries.map(([key, value]) => (
        <div className="contents" key={key}>
          <div className="text-muted-foreground font-medium">{key}</div>
          <div className="truncate">{value}</div>
        </div>
      ))}
    </div>
  );
}

function PoolInfoCard() {
  const {
    data: poolInfo,
    isFetching,
    isLoading,
    refetch,
  } = useUsenetPoolInfo();
  const rebuild = useRebuildUsenetPoolMutation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Connection Pool</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-4">
            <Spinner />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!poolInfo) {
    return null;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Connection Pool</CardTitle>
        <CardDescription>Usenet Connection Pool Status</CardDescription>
        <CardAction className="flex-wrap">
          <ButtonGroup>
            <Button
              disabled={isFetching || rebuild.isPending}
              onClick={() => refetch()}
              size="icon-sm"
              title="Refresh Pool"
              variant="outline"
            >
              <RefreshCwIcon className={cn(isFetching && "animate-spin")} />
            </Button>
            <Button
              disabled={isFetching || rebuild.isPending}
              onClick={() => {
                toast.promise(rebuild.mutateAsync(), {
                  error(err: APIError) {
                    console.error(err);
                    return {
                      closeButton: true,
                      message: err.message,
                    };
                  },
                  loading: "Rebuilding pool...",
                  success: "Pool rebuilt",
                });
              }}
              size="icon-sm"
              title="Rebuild Pool"
              variant="outline"
            >
              <HammerIcon
                className={cn(rebuild.isPending && "animate-bounce")}
              />
            </Button>
          </ButtonGroup>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <div className="flex flex-row flex-wrap justify-between gap-4">
          <div>
            <div className="text-muted-foreground font-medium">Providers</div>
            <div className="mt-1 text-lg font-semibold">
              {poolInfo.total_providers}
            </div>
          </div>
          <div>
            <div className="text-muted-foreground font-medium">Max Conn.</div>
            <div className="mt-1 text-lg font-semibold">
              {poolInfo.max_connections}
            </div>
          </div>
          <div>
            <div className="text-muted-foreground font-medium">
              Active Conn.
            </div>
            <div className="mt-1 text-lg font-semibold">
              {poolInfo.active_connections}
            </div>
          </div>
          <div>
            <div className="text-muted-foreground font-medium">Idle Conn.</div>
            <div className="mt-1 text-lg font-semibold">
              {poolInfo.idle_connections}
            </div>
          </div>
        </div>

        <div>
          <h3 className="mb-3 text-sm font-semibold">Providers</h3>
          <div className="flex flex-col gap-2">
            {!poolInfo.providers.length && (
              <Item className="bg-muted/50 flex items-center gap-4 rounded-md px-3 py-2 text-sm">
                <ItemContent>
                  <ItemTitle>
                    Add a{" "}
                    <Link
                      className="text-primary underline underline-offset-4"
                      to="/dash/usenet/servers"
                    >
                      Usenet Server
                    </Link>
                  </ItemTitle>
                </ItemContent>
              </Item>
            )}
            {poolInfo.providers.map((provider) => (
              <Item
                className="bg-muted/50 flex items-center gap-4 rounded-md px-3 py-2 text-sm"
                key={provider.id}
              >
                <ItemContent>
                  <ItemTitle className="flex w-full flex-row flex-wrap justify-between">
                    <div>{provider.id}</div>

                    <Badge
                      asChild
                      variant={getStateBadgeVariant(provider.state)}
                    >
                      <div>
                        <span
                          className={`inline-block h-1.5 w-1.5 rounded-full ${getStateColor(
                            provider.state,
                          )}`}
                        />
                        {getStateLabel(provider.state)}
                      </div>
                    </Badge>
                  </ItemTitle>
                  <ItemDescription className="flex flex-row flex-wrap gap-4">
                    <div>
                      <span>Priority: {provider.priority}</span>
                    </div>
                    {provider.is_backup && (
                      <div>
                        <span>Backup</span>
                      </div>
                    )}
                    <div>
                      <span>
                        {provider.active_connections}/{provider.max_connections}{" "}
                        active
                      </span>
                    </div>
                    <div>
                      <span>{provider.idle_connections} idle</span>
                    </div>
                  </ItemDescription>
                </ItemContent>
              </Item>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function RouteComponent() {
  const { data: config, isLoading } = useUsenetConfig();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    );
  }

  if (!config) {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Newz Config</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 text-sm">
            {configFields.map(({ key, label }) => (
              <div key={key}>
                <div className="text-muted-foreground font-medium">{label}</div>
                <div className="mt-1">{config[key]}</div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Indexer Request Headers</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          <div>
            <h3 className="mb-3 text-sm font-semibold">Query Headers</h3>
            <div className="flex flex-col gap-4">
              {Object.entries(config.indexer_request_header.query).map(
                ([queryType, headers]) => (
                  <div key={queryType}>
                    <div className="text-foreground mb-1 text-sm font-medium">
                      {queryTypeLabels[queryType] ?? queryType}
                    </div>
                    <HeaderTable headers={headers} />
                  </div>
                ),
              )}
            </div>
          </div>
          <div>
            <h3 className="mb-3 text-sm font-semibold">Grab Headers</h3>
            <HeaderTable headers={config.indexer_request_header.grab} />
          </div>
        </CardContent>
      </Card>
      <PoolInfoCard />
    </div>
  );
}
