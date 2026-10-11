import { describe, expect, it } from "vitest";
import { codeComplete, normalizeCode, qrPayload, stripClaimToken } from "./pairing";

describe("normalizeCode", () => {
  it("formats 8 chars with a dash", () => {
    expect(normalizeCode(" abcd efgh ")).toBe("ABCD-EFGH");
    expect(normalizeCode("deskdrop:kzdn-sufr")).toBe("KZDN-SUFR");
  });
  it("complete is ABCD-EFGH length 9", () => {
    expect(codeComplete("abcd-efgh")).toBe(true);
    expect(codeComplete("abc")).toBe(false);
  });
  it("qr payload uses deskdrop prefix", () => {
    expect(qrPayload("abcd efgh")).toBe("deskdrop:ABCD-EFGH");
  });
});

describe("stripClaimToken", () => {
  it("drops token so the browser never stores the raw device secret", () => {
    expect(stripClaimToken({ token: "secret", device_id: "d1", type: "web" })).toEqual({
      device_id: "d1",
      type: "web",
    });
  });
});
