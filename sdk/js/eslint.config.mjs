import eslint from "@eslint/js";
import perfectionist from "eslint-plugin-perfectionist";
import globals from "globals";
import tseslint from "typescript-eslint";

function getPerfectionistRules(
  config = { "sort-object-types": {}, "sort-objects": {} },
) {
  const defaultRules = {
    "sort-object-types": {
      partitionByNewLine: true,
    },
    "sort-objects": {
      partitionByComment: true,
      partitionByNewLine: true,
    },
  };
  const rules = {};
  for (const [ruleName, ruleConfig] of Object.entries(config)) {
    rules[`perfectionist/${ruleName}`] = [
      "error",
      {
        ...defaultRules[ruleName],
        ...ruleConfig,
      },
    ];
  }
  return rules;
}

export default tseslint.config(
  {
    ignores: ["**/dist/"],
  },
  {
    extends: [
      eslint.configs.recommended,
      perfectionist.configs["recommended-natural"],
    ],
    rules: {
      ...getPerfectionistRules(),
    },
  },
  {
    files: ["**/*.js"],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
  {
    extends: tseslint.configs.recommended,
    files: ["**/*.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
          destructuredArrayIgnorePattern: "^_",
          ignoreRestSiblings: true,
          varsIgnorePattern: "^_",
        },
      ],
    },
  },
);
