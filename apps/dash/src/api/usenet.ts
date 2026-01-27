import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type ParsedNZB = {
  files: ParsedNZBFile[];
  meta: Record<string, string>;
  size: number;
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

async function parseNzbFile(file: File) {
  const formData = new FormData();
  formData.append("file", file);

  const { data } = await api<ParsedNZB>("POST /usenet/nzb/parse", {
    body: formData,
  });
  return data;
}
