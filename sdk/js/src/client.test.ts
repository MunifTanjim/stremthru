import { StremThru } from "./client";

describe("StremThru", () => {
  it("can create instance", () => {
    new StremThru({ auth: "user:pass", baseUrl: "http://127.0.0.1:7860" });
  });
});
