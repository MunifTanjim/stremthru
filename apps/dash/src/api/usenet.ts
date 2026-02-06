import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type ParsedNZB = {
  files: ParsedNZBFile[];
  meta: Record<string, string>;
  size: number;
};

export type UsenetConfig = {
  indexer_request_header: {
    grab: Record<string, string>;
    query: Record<string, Record<string, string>>;
  };
  max_connection_per_stream: number;
  nzb_cache_size: string;
  nzb_cache_ttl: string;
  nzb_max_file_size: string;
  segment_cache_size: string;
  stream_buffer_size: string;
};

type ParsedNZBFile = {
  date: string;
  groups: string[];
  name: string;
  poster: string;
  segments: ParsedNZBFileSegment[];
  size: number;
  subject: string;
};

type ParsedNZBFileSegment = {
  bytes: number;
  message_id: string;
  number: number;
};

export function useNzbParseMutation() {
  return useMutation({
    mutationFn: parseNzbFile,
  });
}

export function useUsenetConfig() {
  return useQuery({
    queryFn: async () => {
      const { data } = await api<UsenetConfig>("/usenet/config");
      return data;
    },
    queryKey: ["/usenet/config"],
    staleTime: Infinity,
  });
}

async function parseNzbFile(file: File) {
  const formData = new FormData();
  formData.append("file", file);

  const { data } = await api<ParsedNZB>("POST /usenet/nzb/parse", {
    body: formData,
  });
  return data;
}
