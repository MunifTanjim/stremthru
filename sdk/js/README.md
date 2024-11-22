# StremThru - JavaScript SDK

## Installation

```sh
# using pnpm:
pnpm add stremthru

# using npm:
npm install --save stremthru

# using yarn:
yarn add stremthru
```

## Usage

**Basic Usage:**

```ts
import { StremThru } from "stremthru";

const st = new StremThru({
  baseUrl: "http://127.0.0.1:8080",
  auth: "user:pass",
});
```

## License

Licensed under the MIT License. Check the [LICENSE](./LICENSE) file for details.
