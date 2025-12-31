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

type DownloadNzbFileParams = {
  groups: string[];
  name: string;
  segments: ParsedNZBFileSegment[];
};

export async function downloadNzbFile(params: DownloadNzbFileParams) {
  const response = await fetch("/dash/api/usenet/nzb/download", {
    body: JSON.stringify(params),
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || "Download failed");
  }

  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = params.name;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
