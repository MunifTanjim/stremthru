import js from "@eslint/js";
import pluginQuery from "@tanstack/eslint-plugin-query";
import perfectionist from "eslint-plugin-perfectionist";
import reactDom from "eslint-plugin-react-dom";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import reactX from "eslint-plugin-react-x";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

export default defineConfig([
  globalIgnores(["dist"]),
  {
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      perfectionist.configs["recommended-natural"],
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parser: tseslint.parser,
      parserOptions: {
        projectService: {
          allowDefaultProject: ["*.js", "apps/dash/*.js"],
        },
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },
  {
    extends: [
      reactHooks.configs["recommended-latest"],
      reactRefresh.configs.vite,
      reactX.configs["recommended-typescript"],
      reactDom.configs.recommended,
      pluginQuery.configs["flat/recommended"],
    ],
    files: ["apps/dash/**/*.{ts,tsx}"],
  },
  {
    files: ["apps/dash/src/components/ui/**"],
    rules: {
      "react-refresh/only-export-components": ["off"],
      ...Object.keys(perfectionist.rules).reduce((rules, key) => {
        rules[`perfectionist/${key}`] = ["off"];
        return rules;
      }, {}),
    },
  },
]);
