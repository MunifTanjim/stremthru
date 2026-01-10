import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type ParsedNZB = {
  files: ParsedNZBFile[];
  meta: Record<string, string>;
  size: number;
};

type DownloadNzbFileParams = {
  groups: string[];
  name: string;
  segments: ParsedNZBFileSegment[];
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

export async function downloadNzbFile(params: DownloadNzbFileParams) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/dash/api/usenet/nzb/download";

  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "nzb_file";
  input.value = JSON.stringify(params);
  form.appendChild(input);

  document.body.appendChild(form);
  form.submit();
  form.remove();
}

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
