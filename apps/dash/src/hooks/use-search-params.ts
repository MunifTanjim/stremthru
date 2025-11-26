import {
  NavigateOptions,
  RegisteredRouter,
  useNavigate,
  useSearch,
} from "@tanstack/react-router";
import { useCallback } from "react";

import { FileRouteTypes } from "@/routeTree.gen";

export function useSearchParams<T extends FileRouteTypes["fullPaths"]>(
  fullPath: T,
) {
  const params = useSearch({ from: fullPath });
  const navigate = useNavigate({ from: fullPath });
  const setParams = useCallback(
    (search: NavigateOptions<RegisteredRouter, T, T, T, T>["search"]) => {
      return navigate({ search });
    },
    [navigate],
  );
  return [params, setParams] as const;
}
