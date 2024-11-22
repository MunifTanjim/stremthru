export type StoreMagnetStatus =
  | "cached"
  | "downloaded"
  | "downloading"
  | "failed"
  | "invalid"
  | "processing"
  | "queued"
  | "unknown"
  | "uploading";

export type StoreUserSubscriptionStatus = "expired" | "premium" | "trial";
