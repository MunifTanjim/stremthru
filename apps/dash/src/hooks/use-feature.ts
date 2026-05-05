import { useMemo } from "react";

import { useConfig } from "@/api/config";

export function useFeature(): ReadonlyMap<string, boolean> & {
  isReady: boolean;
} {
  const { data } = useConfig();
  return useMemo(() => {
    const map = new Map<string, boolean>();
    for (const f of data?.features ?? []) {
      map.set(f.name, f.enabled);
    }
    return Object.assign(map, { isReady: !!data });
  }, [data]);
}
