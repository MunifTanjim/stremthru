import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type IMDBTitle = {
  id: string;
  title: string;
  type: string;
  year: number;
};

export function useIMDBAutocomplete(query: string = "") {
  return useQuery({
    enabled: Boolean(query),
    queryFn: async () => {
      const { data } = await api<IMDBTitle[]>(
        `/imdb/autocomplete?query=${encodeURIComponent(query)}`,
      );
      return data;
    },
    queryKey: ["/imdb/autocomplete", query],
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
