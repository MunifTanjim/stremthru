import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { api, APIError } from "@/lib/api";

export type AuthedUser = {
  id: string;
};

export function useAuthedUser() {
  const queryFn = useCallback(async () => {
    try {
      return await getAuthedUser();
    } catch (err) {
      if (err instanceof APIError && err.status == 401) {
        return null;
      }
      throw err;
    }
  }, []);
  return useQuery({
    queryFn,
    queryKey: ["/auth/user"],
    refetchOnWindowFocus: true,
    retry: false,
  });
}

async function getAuthedUser() {
  const { data } = await api<AuthedUser>("/auth/user");
  return data;
}

async function signInWithPassword(body: { password: string; user: string }) {
  const { data } = await api<AuthedUser>("/auth/signin", {
    body,
    method: "POST",
  });
  return data;
}

export const signIn = {
  password: signInWithPassword,
} as const;

export async function signOut() {
  await api<null>("/auth/signout", { method: "POST" });
}

export function useSignIn<M extends keyof typeof signIn>(method: M) {
  const queryClient = useQueryClient();
  const mutationFn = signIn[method];

  return useMutation({
    mutationFn,
    onSuccess: (data) => {
      queryClient.setQueryData(["/auth/user"], data);
      queryClient.invalidateQueries({ queryKey: ["/auth/user"] });
    },
  });
}

export function useSignOut() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: signOut,
    onSuccess: () => {
      queryClient.setQueryData(["/auth/user"], null);
      queryClient.invalidateQueries({ queryKey: ["/auth/user"] });
    },
  });
}
