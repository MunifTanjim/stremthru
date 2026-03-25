import { createFileRoute } from "@tanstack/react-router";
import { DateTime } from "luxon";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import {
  useMaintenanceMutation,
  useMaintenanceStatus,
} from "@/api/maintenance";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { extractErrorMessages } from "@/lib/api";

export const Route = createFileRoute("/dash/settings/maintenance")({
  component: RouteComponent,
  staticData: {
    crumb: "Maintenance",
  },
});

function RouteComponent() {
  const status = useMaintenanceStatus();
  const { activate, deactivate } = useMaintenanceMutation();
  const [duration, setDuration] = useState("60s");
  const [timeLeft, setTimeLeft] = useState("");

  useEffect(() => {
    if (!status.data?.is_active || !status.data.ends_at) {
      setTimeLeft("");
      return;
    }

    const endsAt = DateTime.fromISO(status.data.ends_at);

    const update = () => {
      const diff = endsAt.diffNow(["minutes", "seconds"]);
      if (diff.toMillis() <= 0) {
        setTimeLeft("Expired");
        return;
      }
      const mins = Math.floor(diff.minutes);
      const secs = Math.floor(diff.seconds);
      setTimeLeft(mins > 0 ? `${mins}m ${secs}s` : `${secs}s`);
    };

    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [status.data?.is_active, status.data?.ends_at]);

  const handleActivate = () => {
    toast.promise(activate.mutateAsync(duration ? { duration } : undefined), {
      error(err: unknown) {
        return extractErrorMessages(err).join(", ") || "Failed to activate";
      },
      loading: "Activating maintenance...",
      success: "Maintenance activated!",
    });
  };

  const handleDeactivate = () => {
    toast.promise(deactivate.mutateAsync(), {
      error(err: unknown) {
        return extractErrorMessages(err).join(", ") || "Failed to deactivate";
      },
      loading: "Deactivating maintenance...",
      success: "Maintenance deactivated!",
    });
  };

  if (status.isLoading) {
    return <div className="text-muted-foreground text-sm">Loading...</div>;
  }

  const isActive = status.data?.is_active ?? false;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Maintenance</h2>
        <Badge variant={isActive ? "destructive" : "secondary"}>
          {isActive ? "Active" : "Inactive"}
        </Badge>
      </div>

      {isActive && (
        <div className="flex flex-col gap-4">
          <div className="text-muted-foreground text-sm">
            Maintenance mode is active.
            {timeLeft && (
              <span>
                {" "}
                Time remaining: <strong>{timeLeft}</strong>
              </span>
            )}
          </div>
          <div>
            <Button
              disabled={deactivate.isPending}
              onClick={handleDeactivate}
              variant="destructive"
            >
              Deactivate Maintenance
            </Button>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-4">
        <div className="flex items-end gap-3">
          <div className="flex flex-col gap-2">
            <Label htmlFor="duration">Duration</Label>
            <Input
              id="duration"
              onChange={(e) => setDuration(e.target.value)}
              placeholder="e.g., 30s"
              value={duration}
            />
          </div>
          <Button disabled={activate.isPending} onClick={handleActivate}>
            {isActive ? "Extend Maintenance" : "Activate Maintenance"}
          </Button>
        </div>
        <p className="text-muted-foreground text-sm">
          Activates maintenance mode. Defaults to 60s if no duration is
          specified.
        </p>
      </div>
    </div>
  );
}
