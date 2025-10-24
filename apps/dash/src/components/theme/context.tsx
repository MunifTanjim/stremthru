import React, { createContext, use } from "react";

export type Theme = "dark" | "light" | "system";

type ThemeProviderState = {
  setTheme: (theme: React.SetStateAction<Theme>) => void;
  theme: Theme;
};

const initialState: ThemeProviderState = {
  setTheme: () => null,
  theme: "system",
};

export const ThemeProviderContext =
  createContext<ThemeProviderState>(initialState);

export function useThemeProviderContext() {
  return use(ThemeProviderContext);
}
