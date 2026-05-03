export default {
  "*.{js,jsx,ts,tsx}": ["prettier --write", "eslint --cache --fix"],
  "*.{json,md,yml}": "prettier --write",
  "*.{ts,tsx}": () => "tsc --noEmit",
};
