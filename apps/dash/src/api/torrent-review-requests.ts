import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

import { api } from "@/lib/api";

export type FileCorrection = {
  episode: number;
  path: string;
  prev_episode: number;
  prev_season: number;
  season: number;
};

export type MappingReviewRequest = {
  comment?: string;
  files?: FileCorrection[];
  suggested_mappings?: SuggestedMapping[];
  hash: string;
  mapping_id: string;
  prev_id: string;
  reason: ReviewReason;
  target: MappingTarget;
};

export type MappingTarget = "anidb" | "imdb";

export type SuggestedMapping = {
  s_type: "abs" | "tv" | "ani";
  s: number;
  ep_start: number;
  ep_end: number;
};

export type AniDBMappingPreview = {
  tid: string;
  s_type: "abs" | "tv" | "ani";
  s: number;
  ep_start: number;
  ep_end: number;
};

export type PreviewMappingRequest = {
  anidb_id: string;
};

export type PreviewMappingResponse = {
  mappings: AniDBMappingPreview[];
  anidb_titles: string[];
};

export type ResolveReviewRequestParams = {
  id: number;
  mapping_id?: string;
  mappings?: AniDBMappingPreview[];
};

export type ReviewReason =
  | "fake_torrent"
  | "incomplete_season_pack"
  | "other"
  | "wrong_mapping";

export type ReviewRequest = {
  comment?: string;
  created_at: string;
  files?: FileCorrection[];
  suggested_mappings?: SuggestedMapping[];
  hash: string;
  hash_title?: string;
  id: number;
  mapping_id: string;
  mapping_id_titles?: string[];
  prev_id: string;
  prev_id_titles?: string[];
  reason: ReviewReason;
  resolved_at?: string;
  status: ReviewStatus;
  target: MappingTarget;
};

export type ReviewRequestsParams = {
  limit?: number;
  status?: ReviewStatus;
  target?: MappingTarget;
};

export type ReviewStatus = "pending" | "rejected" | "resolved";

type ReviewRequestsListResponse = {
  items: ReviewRequest[];
  next_cursor: string;
};

export function usePreviewMapping() {
  return useMutation({
    mutationFn: ({ id, params }: { id: number; params: PreviewMappingRequest }) =>
      previewMapping(id, params),
  });
}

export function useRejectReviewRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => rejectReviewRequest(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/torrent/review-requests"] });
    },
  });
}

export function useResolveReviewRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: resolveReviewRequest,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["/torrent/review-requests"],
      });
    },
  });
}

export function useReviewRequests(params: ReviewRequestsParams) {
  const { limit, status, target } = params;

  return useInfiniteQuery<
    ReviewRequestsListResponse,
    Error,
    { pages: ReviewRequestsListResponse[] },
    unknown[],
    string
  >({
    queryKey: ["/torrent/review-requests", { limit, status, target }],
    queryFn: ({ pageParam }) =>
      getReviewRequests({ cursor: pageParam, limit, status, target }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });
}

export function useSubmitMappingReview() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: submitMappingReview,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/torrents/info/imdb"] });
      queryClient.invalidateQueries({ queryKey: ["/torrents/info/anidb"] });
    },
  });
}

async function getReviewRequests(params: {
  cursor?: string;
  limit?: number;
  status?: ReviewStatus;
  target?: MappingTarget;
}) {
  const searchParams = new URLSearchParams();
  if (params.status !== undefined) {
    searchParams.set("status", params.status);
  }
  if (params.target !== undefined) {
    searchParams.set("target", params.target);
  }
  if (params.limit !== undefined) {
    searchParams.set("limit", params.limit.toString());
  }
  if (params.cursor) {
    searchParams.set("cursor", params.cursor);
  }

  const query = searchParams.toString();
  const endpoint =
    `/torrent/review-requests${query ? `?${query}` : ""}` as const;
  const { data } = await api<ReviewRequestsListResponse>(endpoint, {});
  return data;
}

async function resolveReviewRequest(params: ResolveReviewRequestParams) {
  const { id, ...body } = params;
  const { data } = await api<Record<string, never>>(
    `PATCH /torrent/review-requests/${id}/resolve`,
    { body },
  );
  return data;
}

async function previewMapping(
  id: number,
  params: PreviewMappingRequest,
): Promise<PreviewMappingResponse> {
  const { data } = await api<PreviewMappingResponse>(
    `POST /torrent/review-requests/${id}/preview`,
    { body: params },
  );
  return data;
}

async function rejectReviewRequest(id: number): Promise<void> {
  await api(`POST /torrent/review-requests/${id}/reject`, {
    body: {},
  });
}

async function submitMappingReview(params: MappingReviewRequest) {
  const { data } = await api<void>("POST /torrent/review-requests", {
    body: params,
  });
  return data;
}
