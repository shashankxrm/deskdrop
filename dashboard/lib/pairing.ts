export function normalizeCode(raw: string): string {
  const stripped = raw.trim().replace(/^deskdrop:/i, "");
  let out = "";
  for (const ch of stripped.toUpperCase()) {
    if (ch === "-" || ch === " ") continue;
    if ((ch >= "A" && ch <= "Z") || (ch >= "0" && ch <= "9")) out += ch;
  }
  if (out.length === 8) return `${out.slice(0, 4)}-${out.slice(4)}`;
  return out;
}

export function codeComplete(raw: string): boolean {
  return normalizeCode(raw).length === 9;
}

export function qrPayload(code: string): string {
  return `deskdrop:${normalizeCode(code)}`;
}

export function stripClaimToken(body: unknown): unknown {
  if (!body || typeof body !== "object" || Array.isArray(body)) return body;
  const copy = { ...(body as Record<string, unknown>) };
  delete copy.token;
  return copy;
}
