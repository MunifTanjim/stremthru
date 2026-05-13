import { useMutation, useQueryClient } from "@tanstack/react-query";

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
  hash: string;
  mapping_id: string;
  prev_id: string;
  reason: ReviewReason;
  target: MappingTarget;
};

export type MappingTarget = "anidb" | "imdb";

export type ReviewReason =
  | "fake_torrent"
  | "incomplete_season_pack"
  | "other"
  | "wrong_mapping";

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

async function submitMappingReview(params: MappingReviewRequest) {
  const { data } = await api<void>("/torrent/mapping/review", {
    body: params,
    method: "POST",
  });
  return data;
}
