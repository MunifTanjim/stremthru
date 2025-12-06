import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type CreateTraktAccountParams = {
  oauth_token_id: string;
};

export type TraktAccount = {
  created_at: string;
  id: string; // trakt user slug
  is_valid: boolean;
  updated_at: string;
  user_name: string;
};

export type TraktAuthURL = {
  url: string;
};

export async function getTraktAuthURL(state: string) {
  const { data } = await api<TraktAuthURL>(
    `/vault/trakt/auth/url?state=${state}`,
  );
  return data.url;
}

export function useTraktAccountMutation() {
  const create = useMutation({
    mutationFn: createTraktAccount,
    onSuccess: async (_, __, ___, ctx) => {
      await ctx.client.invalidateQueries({
        queryKey: ["/vault/trakt/accounts"],
      });
    },
  });

  const remove = useMutation({
    mutationFn: deleteTraktAccount,
    onSuccess: async (_, id, __, ctx) => {
      const list = ctx.client.getQueryData<TraktAccount[]>([
        "/vault/trakt/accounts",
      ]);
      if (list) {
        ctx.client.setQueryData(
          ["/vault/trakt/accounts"],
          list.filter((item) => item.id !== id),
        );
      }
    },
  });

  return { create, remove };
}

export function useTraktAccounts() {
  return useQuery({
    queryFn: getTraktAccounts,
    queryKey: ["/vault/trakt/accounts"],
  });
}

async function createTraktAccount(params: CreateTraktAccountParams) {
  const { data } = await api<TraktAccount>("POST /vault/trakt/accounts", {
    body: params,
  });
  return data;
}

async function deleteTraktAccount(id: string) {
  await api(`DELETE /vault/trakt/accounts/${id}`);
}

async function getTraktAccounts() {
  const { data } = await api<TraktAccount[]>("/vault/trakt/accounts");
  return data;
}
