export const sessionCookieName = "deskdrop_device";

export const sessionCookieAttrs = {
  httpOnly: true,
  path: "/",
  sameSite: "lax" as const,
  secure: false,
  maxAge: 365 * 24 * 3600,
};

export function tokenFromClaimBody(body: unknown): string {
  if (!body || typeof body !== "object" || Array.isArray(body)) return "";
  const t = (body as { token?: unknown }).token;
  return typeof t === "string" ? t : "";
}
