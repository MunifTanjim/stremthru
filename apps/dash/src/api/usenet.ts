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

export type UsenetPoolInfo = {
  active_connections: number;
  idle_connections: number;
  max_connections: number;
  providers: UsenetPoolProviderInfo[];
  total_providers: number;
};

export type UsenetPoolProviderInfo = {
  active_connections: number;
  id: string;
  idle_connections: number;
  is_backup: boolean;
  max_connections: number;
  priority: number;
  state: "auth_failed" | "connecting" | "disabled" | "offline" | "online";
  total_connections: number;
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

export function useNzbUploadMutation() {
  return useMutation({
    mutationFn: uploadNzbFile,
  });
}

export function useRebuildUsenetPoolMutation() {
  return useMutation({
    mutationFn: async () => {
      const { data } = await api<UsenetPoolInfo>("POST /usenet/pool/rebuild");
      return data;
    },
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({ queryKey: ["/usenet/pool"] });
    },
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

export function useUsenetPoolInfo() {
  return useQuery({
    queryFn: async () => {
      const { data } = await api<UsenetPoolInfo>("/usenet/pool");
      return data;
    },
    queryKey: ["/usenet/pool"],
    refetchInterval: 10_000,
    staleTime: 5_000,
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

async function uploadNzbFile({ file, name }: { file: File; name: string }) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("name", name);

  const { data } = await api<{ id: string }>("POST /usenet/nzb/upload", {
    body: formData,
  });
  return data;
}
