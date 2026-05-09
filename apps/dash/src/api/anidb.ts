import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type AniDBTitle = {
  id: string;
  season: string;
  title: string;
  type: string;
  year: string;
};

export function useAniDBAutocomplete(query: string = "") {
  return useQuery({
    enabled: Boolean(query),
    queryFn: async () => {
      const { data } = await api<AniDBTitle[]>(
        `/anidb/autocomplete?query=${encodeURIComponent(query)}`,
      );
      return data;
    },
    queryKey: ["/anidb/autocomplete", query],
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
