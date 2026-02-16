import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";

type ProxifyLinkParams = {
  encrypt?: boolean;
  exp?: string;
  filename?: string;
  req_headers?: string;
  url: string;
};

type ProxifyLinkResult = {
  url: string;
};

export function useProxifyLinkMutation() {
  return useMutation({
    mutationFn: proxifyLink,
  });
}

function proxifyLink(params: ProxifyLinkParams) {
  return api<ProxifyLinkResult>("POST /proxy", { body: params });
}
