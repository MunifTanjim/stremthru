import { createContext, use } from "react";

import type { AuthedUser } from "@/api/auth";

export type AuthContextValue = { refresh: () => Promise<void> } & (
  | { status: "authed"; user: AuthedUser }
  | { status: "loading" | "unauthed"; user: null }
);

export const AuthContext = createContext<AuthContextValue>({
  refresh: async () => {},
  status: "loading",
  user: null,
});

export function useCurrentAuth() {
  return use(AuthContext);
}

export function useCurrentUser() {
  const { user } = use(AuthContext);
  if (!user) {
    throw new Error(
      "useCurrentUser should be called inside authenticated context",
    );
  }
  return user;
}
