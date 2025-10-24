import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useRef } from "react";

import { useAuthedUser } from "@/api/auth";
import { Spinner } from "@/components/ui/spinner";

import { AuthContext, AuthContextValue } from "./context";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const isInitialLoadDone = useRef(false);

  const user = useAuthedUser();

  const queryClient = useQueryClient();
  const refresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["/auth/user"] });
  }, [queryClient]);

  const value = useMemo<AuthContextValue>(() => {
    if (user.data) {
      isInitialLoadDone.current = true;
      return {
        refresh,
        status: "authed",
        user: user.data,
      };
    }

    if (!user.isLoading) {
      isInitialLoadDone.current = true;
      return {
        refresh,
        status: "unauthed",
        user: null,
      };
    }

    return {
      refresh,
      status: "loading",
      user: null,
    };
  }, [refresh, user.data, user.isLoading]);

  return (
    <AuthContext value={value}>
      {isInitialLoadDone.current ? (
        children
      ) : (
        <div className="flex h-screen w-full items-center justify-center">
          <Spinner className="size-14" />
        </div>
      )}
    </AuthContext>
  );
}
