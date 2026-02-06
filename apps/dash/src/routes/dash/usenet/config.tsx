import { createFileRoute } from "@tanstack/react-router";

import { type UsenetConfig, useUsenetConfig } from "@/api/usenet";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";

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
    </div>
  );
}
