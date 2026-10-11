import { describe, expect, it } from "vitest";
import { tokenFromClaimBody } from "./session";

describe("tokenFromClaimBody", () => {
  it("reads token before the proxy strips it", () => {
    expect(tokenFromClaimBody({ token: "abc", device_id: "d" })).toBe("abc");
    expect(tokenFromClaimBody({ device_id: "d" })).toBe("");
  });
});
