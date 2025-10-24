export default {
  "*.{js,jsx,ts,tsx}": ["prettier --write", "eslint"],
  "*.{json,md,yml}": "prettier --write",
  "*.{ts,tsx}": () => "tsc --noEmit",
};
