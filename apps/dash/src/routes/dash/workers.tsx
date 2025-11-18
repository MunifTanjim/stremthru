import { createFileRoute } from "@tanstack/react-router";
import { ColumnDef } from "@tanstack/react-table";
import { DateTime, Duration } from "luxon";
import { useEffect, useMemo, useState } from "react";

import {
  useWorkerDetails,
  useWorkerJobLogs,
  WorkerJobLog,
} from "@/api/workers";
import { DataTable } from "@/components/data-table";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export const Route = createFileRoute("/dash/workers")({
  component: RouteComponent,
  staticData: {
    crumb: "Workers",
  },
});

const jobLogsColumns: ColumnDef<WorkerJobLog>[] = [
  {
    accessorKey: "id",
    header: "ID",
  },
  {
    accessorKey: "created_at",
    cell: ({ getValue }) => {
      const date = DateTime.fromISO(getValue<string>());
      return date.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS);
    },
    header: "Started At",
  },
  {
    accessorKey: "status",
    cell: ({ getValue }) => {
      const status = getValue<string>();
      const colors = {
        done: "text-green-500",
        failed: "text-red-500",
        started: "text-cyan-500",
      };
      return (
        <span className={colors[status as keyof typeof colors] || ""}>
          {status}
        </span>
      );
    },
    header: "Status",
  },
  {
    accessorKey: "updated_at",
    cell: ({ getValue }) => {
      const date = DateTime.fromISO(getValue<string>());
      return date.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS);
    },
    header: "Last Heartbeat At",
  },
  {
    accessorKey: "error",
    cell: ({ getValue }) => {
      const error = getValue<string | undefined>();
      return error ? (
        <span className="font-mono text-xs text-red-600">{error}</span>
      ) : (
        "-"
      );
    },
    header: "Error",
  },
];

function RouteComponent() {
  const workerDetails = useWorkerDetails();

  const [selectedWorkerId, setSelectedWorkerId] = useState("");

  const jobLogs = useWorkerJobLogs(selectedWorkerId);

  const workerOptions = useMemo(() => {
    return Object.entries(workerDetails.data ?? {})
      .map(([value, details]) => ({
        indicator: details.has_failed_job ? `❗` : "",
        label: details.title,
        value,
      }))
      .sort((a, b) => a.label.localeCompare(b.label))
      .map((o) => ({
        ...o,
        label: `${o.indicator ? `${o.indicator} ` : ""}${o.label}`,
      }));
  }, [workerDetails.data]);

  useEffect(() => {
    setSelectedWorkerId((workerId) => {
      if (workerId || !workerOptions.length) {
        return workerId;
      }
      return workerOptions[0].value;
    });
  }, [workerOptions]);

  const selectedWorkerInterval = useMemo(() => {
    const worker = workerDetails.data?.[selectedWorkerId];
    if (!worker) {
      return "";
    }
    return Duration.fromMillis(worker.interval / 1000 / 1000)
      .shiftTo("months", "days", "hours", "minutes", "seconds")
      .removeZeros()
      .toHuman({ maximumFractionDigits: 0 });
  }, [selectedWorkerId, workerDetails.data]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-4">
        <Label className="text-sm font-medium" htmlFor="worker">
          Worker:
        </Label>
        {workerDetails.isLoading ? (
          <div className="text-muted-foreground text-sm">
            Loading workers...
          </div>
        ) : workerDetails.isError ? (
          <div className="text-sm text-red-600">Error loading workers</div>
        ) : (
          <Select onValueChange={setSelectedWorkerId} value={selectedWorkerId}>
            <SelectTrigger className="w-[300px]" id="worker">
              <SelectValue placeholder="Select worker" />
            </SelectTrigger>
            <SelectContent>
              {workerOptions.map(({ label, value }) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <div>
          {selectedWorkerInterval && (
            <div>Interval: {selectedWorkerInterval}</div>
          )}
        </div>
      </div>

      <div>
        <h3 className="text-md mb-4 font-semibold">Job Logs</h3>
        {selectedWorkerId &&
          (jobLogs.isLoading ? (
            <div className="text-muted-foreground text-sm">
              Loading job logs...
            </div>
          ) : jobLogs.isError ? (
            <div className="text-sm text-red-600">Error loading job logs</div>
          ) : (
            <DataTable columns={jobLogsColumns} data={jobLogs.data || []} />
          ))}
      </div>
    </div>
  );
}
