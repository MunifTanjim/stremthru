import { createFileRoute, Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";
import { DateTime } from "luxon";
import { useState } from "react";

import { type ConfigData, useConfig } from "@/api/config";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/dash/settings/config")({
  component: RouteComponent,
  staticData: {
    crumb: "Config",
  },
});

function CollapsibleConfigSection({
  children,
  gradient,
  icon,
  settingsCount,
  title,
}: {
  children: React.ReactNode;
  gradient: string;
  icon: string;
  settingsCount: number;
  title: string;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Collapsible onOpenChange={setOpen} open={open}>
      <Card>
        <CollapsibleTrigger className="w-full cursor-pointer">
          <CardHeader>
            <div className="flex items-center gap-3">
              <SectionIcon gradient={gradient} letter={icon} />
              <CardTitle>{title}</CardTitle>
              <div className="ml-auto flex items-center gap-2">
                <span className="text-muted-foreground text-xs">
                  {settingsCount} settings
                </span>
                <ChevronRight
                  className={`text-muted-foreground size-4 transition-transform duration-200 ${open ? "rotate-90" : ""}`}
                />
              </div>
            </div>
          </CardHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className="flex flex-col gap-0">{children}</CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}

function ConfigEntry({
  className,
  label,
  value,
}: {
  className?: string;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className={cn("py-2", className)}>
      <div className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
        {label}
      </div>
      <div className="mt-1 break-all font-mono text-sm">{value}</div>
    </div>
  );
}

function ConfigSection({
  children,
  gradient,
  headerExtra,
  icon,
  title,
}: {
  children?: React.ReactNode;
  gradient: string;
  headerExtra?: React.ReactNode;
  icon: string;
  title: string;
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <SectionIcon gradient={gradient} letter={icon} />
          <CardTitle>{title}</CardTitle>
          {headerExtra && <div className="ml-auto">{headerExtra}</div>}
        </div>
      </CardHeader>
      {children && (
        <CardContent className="flex flex-col gap-0">{children}</CardContent>
      )}
    </Card>
  );
}

function DatabaseSection({ database }: { database: ConfigData["database"] }) {
  return (
    <ConfigSection
      gradient="from-amber-500 to-yellow-500"
      icon="D"
      title="Database"
    >
      <ConfigEntry label="URI" value={database.uri} />
      {database.replica_uris?.map((uri, i) => (
        <ConfigEntry key={uri} label={`Replica ${i + 1}`} value={uri} />
      ))}
    </ConfigSection>
  );
}

function FeatureItem({
  enabled,
  name,
  settings,
}: {
  enabled: boolean;
  name: string;
  settings?: null | Record<string, string>;
}) {
  return (
    <MiniCard>
      <div className="flex items-center gap-2">
        <StatusDot enabled={enabled} />
        <span
          className={`font-mono text-sm font-medium ${!enabled ? "text-muted-foreground" : ""}`}
        >
          {name}
        </span>
        <Badge
          className="ml-auto text-[10px]"
          variant={enabled ? "default" : "outline"}
        >
          {enabled ? "enabled" : "disabled"}
        </Badge>
      </div>
      {enabled &&
        settings &&
        Object.entries(settings).map(([key, value]) => (
          <div className="ml-5 mt-2" key={key}>
            <ConfigEntry label={key} value={value} />
          </div>
        ))}
    </MiniCard>
  );
}

function FeaturesSection({ features }: { features: ConfigData["features"] }) {
  return (
    <ConfigSection
      gradient="from-emerald-500 to-green-600"
      icon="F"
      title="Features"
    >
      <div className="flex flex-col gap-2">
        {features.map((f) => (
          <FeatureItem
            enabled={f.enabled}
            key={f.name}
            name={f.name}
            settings={f.settings}
          />
        ))}
      </div>
    </ConfigSection>
  );
}

function IntegrationsSection({
  integrations,
}: {
  integrations: ConfigData["integrations"];
}) {
  return (
    <ConfigSection
      gradient="from-purple-500 to-violet-500"
      icon="I"
      title="Integrations"
    >
      <div className="flex flex-col gap-2">
        {integrations.map((i) => (
          <FeatureItem
            enabled={i.enabled}
            key={i.name}
            name={i.name}
            settings={i.settings}
          />
        ))}
      </div>
    </ConfigSection>
  );
}

function MiniCard({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-muted/30 border-border/60 rounded-lg border p-3">
      {children}
    </div>
  );
}

function NetworkSection({ network }: { network: ConfigData["network"] }) {
  return (
    <ConfigSection
      gradient="from-blue-500 to-blue-700"
      icon="N"
      title="Network"
    >
      <ConfigEntry label="Machine IP" value={network.machine_ip} />
      {network.tunnel_ips &&
        Object.entries(network.tunnel_ips).map(([proxyHost, ip]) => (
          <ConfigEntry
            key={proxyHost}
            label={`Tunnel IP [${proxyHost}]`}
            value={ip || "(unresolved)"}
          />
        ))}
      {network.buddy_url && (
        <ConfigEntry label="Buddy URL" value={network.buddy_url} />
      )}
      {network.peer_url && (
        <ConfigEntry
          label={`Peer URL${network.peer_flags ? ` (${network.peer_flags})` : ""}`}
          value={network.peer_url}
        />
      )}
      {network.pull_peer_url && (
        <ConfigEntry label="Pull Peer URL" value={network.pull_peer_url} />
      )}
    </ConfigSection>
  );
}

function NewzSection({ newz }: { newz: ConfigData["newz"] }) {
  if (newz.disabled) return null;

  return (
    <ConfigSection
      gradient="from-orange-500 to-amber-500"
      headerExtra={
        <Link
          className="text-primary text-sm hover:underline"
          to="/dash/usenet/config"
        >
          View Config
        </Link>
      }
      icon="N"
      title="Newz"
    />
  );
}

function RouteComponent() {
  const { data, isLoading } = useConfig();

  if (isLoading) {
    return <div className="text-muted-foreground text-sm">Loading...</div>;
  }

  if (!data) {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <ServerSection server={data.server} />
      <ConfigSection
        gradient="from-slate-500 to-gray-500"
        icon="I"
        title="Instance"
      >
        <ConfigEntry label="Instance ID" value={data.instance.id} />
        <ConfigEntry label="Base URL" value={data.instance.base_url} />
        <ConfigEntry
          label="Public Instance"
          value={data.instance.is_public_instance ? "true" : "false"}
        />
        <ConfigEntry
          label="Stremio Locked"
          value={data.instance.stremio_locked ? "true" : "false"}
        />
      </ConfigSection>
      <NetworkSection network={data.network} />
      <TunnelSection tunnel={data.tunnel} />
      <DatabaseSection database={data.database} />
      {!data.redis.disabled && (
        <ConfigSection
          gradient="from-red-500 to-orange-500"
          icon="R"
          title="Redis"
        >
          <ConfigEntry label="URI" value={data.redis.uri} />
        </ConfigSection>
      )}
      <UsersSection users={data.users} />
      <StoresSection stores={data.stores} />
      <FeaturesSection features={data.features} />
      <IntegrationsSection integrations={data.integrations} />
      <NewzSection newz={data.newz} />
      <TorzSection torz={data.torz} />
    </div>
  );
}

function SectionIcon({
  gradient,
  letter,
}: {
  gradient: string;
  letter: string;
}) {
  return (
    <div
      className={`flex size-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br text-sm font-bold text-white ${gradient}`}
    >
      {letter}
    </div>
  );
}

function ServerSection({ server }: { server: ConfigData["server"] }) {
  const startedAt = DateTime.fromISO(server.started_at);
  return (
    <ConfigSection
      gradient="from-indigo-500 to-violet-500"
      headerExtra={
        <Badge className="text-xs" variant="secondary">
          {server.version}
        </Badge>
      }
      icon="S"
      title="Server"
    >
      {server.environment && (
        <ConfigEntry label="Environment" value={server.environment} />
      )}
      <ConfigEntry label="Listen Address" value={server.listen_addr} />
      <ConfigEntry
        label="Started At"
        value={startedAt.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS)}
      />
      <ConfigEntry label="Log Level" value={server.log_level} />
      <ConfigEntry label="Log Format" value={server.log_format} />
      <ConfigEntry label="Data Directory" value={server.data_dir} />
      <ConfigEntry
        label="Posthog"
        value={server.posthog_enabled ? "enabled" : "disabled"}
      />
    </ConfigSection>
  );
}

function StatusDot({ enabled }: { enabled: boolean }) {
  return (
    <div
      className={`size-2 shrink-0 rounded-full ${
        enabled
          ? "bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.4)]"
          : "bg-muted-foreground/40"
      }`}
    />
  );
}

function StoresSection({ stores }: { stores: ConfigData["stores"] }) {
  return (
    <ConfigSection
      gradient="from-teal-500 to-emerald-500"
      icon="S"
      title="Stores"
    >
      <ConfigEntry
        className="mb-4"
        label="Store Client User Agent"
        value={stores.client_user_agent}
      />

      <div className="flex flex-col gap-2">
        {stores.items.map((s) => (
          <MiniCard key={s.name}>
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-medium">{s.name}</span>
              {s.config && (
                <span className="text-muted-foreground text-xs">
                  ({s.config})
                </span>
              )}
            </div>
            {(s.cached_stale_time || s.uncached_stale_time) && (
              <div className="ml-2 mt-2 flex gap-4">
                {s.cached_stale_time && (
                  <ConfigEntry
                    label="Cached Stale Time"
                    value={s.cached_stale_time}
                  />
                )}
                {s.uncached_stale_time && (
                  <ConfigEntry
                    label="Uncached Stale Time"
                    value={s.uncached_stale_time}
                  />
                )}
              </div>
            )}
          </MiniCard>
        ))}
      </div>
    </ConfigSection>
  );
}

function TorzSection({ torz }: { torz: ConfigData["torz"] }) {
  const settingsCount = Object.keys(torz).length;

  return (
    <CollapsibleConfigSection
      gradient="from-red-500 to-rose-500"
      icon="T"
      settingsCount={settingsCount}
      title="Torz"
    >
      {torz.torrent_file_cache_size && (
        <ConfigEntry
          label="Torrent File Cache Size"
          value={torz.torrent_file_cache_size}
        />
      )}
      {torz.torrent_file_cache_ttl && (
        <ConfigEntry
          label="Torrent File Cache TTL"
          value={torz.torrent_file_cache_ttl}
        />
      )}
      <ConfigEntry
        label="Torrent File Max Size"
        value={torz.torrent_file_max_size}
      />
    </CollapsibleConfigSection>
  );
}

function TunnelSection({ tunnel }: { tunnel: ConfigData["tunnel"] }) {
  if (tunnel.disabled) return null;
  return (
    <ConfigSection gradient="from-sky-500 to-cyan-500" icon="T" title="Tunnel">
      <ConfigEntry label="Default" value={tunnel.default} />
      {tunnel.by_host &&
        Object.entries(tunnel.by_host).map(([host, proxy]) => (
          <ConfigEntry key={host} label={host} value={proxy} />
        ))}
    </ConfigSection>
  );
}

function UsersSection({ users }: { users: ConfigData["users"] }) {
  if (!users || users.length === 0) return null;
  return (
    <ConfigSection gradient="from-pink-500 to-rose-500" icon="U" title="Users">
      <div className="flex flex-col gap-2">
        {users.map((user) => (
          <MiniCard key={user.name}>
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-medium">{user.name}</span>
              {user.is_admin && <Badge variant="secondary">admin</Badge>}
            </div>
            <div className="ml-2 mt-2">
              <ConfigEntry label="Stores" value={user.stores.join(", ")} />
              {(user.content_proxy_connection_limit ?? 0) > 0 && (
                <ConfigEntry
                  label="Content Proxy Connection Limit"
                  value={user.content_proxy_connection_limit}
                />
              )}
            </div>
          </MiniCard>
        ))}
      </div>
    </ConfigSection>
  );
}
