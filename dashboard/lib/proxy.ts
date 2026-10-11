import { NextResponse } from "next/server";
import { stripClaimToken } from "./pairing";
import { sessionCookieAttrs, sessionCookieName, tokenFromClaimBody } from "./session";

const hopByHop = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailers",
  "transfer-encoding",
  "upgrade",
  "host",
  "content-length",
]);

export function apiBase(): string {
  return (process.env.API_BASE ?? "http://127.0.0.1:8080").replace(/\/$/, "");
}

export async function proxyToAPI(req: Request, path: string[]): Promise<Response> {
  const incoming = new URL(req.url);
  const target = `${apiBase()}/v1/${path.join("/")}${incoming.search}`;
  const headers = new Headers();
  req.headers.forEach((v, k) => {
    if (!hopByHop.has(k.toLowerCase())) headers.set(k, v);
  });
  const method = req.method.toUpperCase();
  const init: RequestInit = { method, headers, redirect: "manual" };
  if (method !== "GET" && method !== "HEAD") {
    init.body = await req.arrayBuffer();
  }
  const upstream = await fetch(target, init);
  const outHeaders = new Headers();
  upstream.headers.forEach((v, k) => {
    if (k.toLowerCase() === "set-cookie") return;
    if (!hopByHop.has(k.toLowerCase())) outHeaders.set(k, v);
  });

  const isClaim = path.join("/") === "pairings/claim";
  const ctype = upstream.headers.get("content-type") ?? "";
  if (isClaim && ctype.includes("application/json")) {
    const raw = await upstream.text();
    let body: unknown = raw;
    try {
      body = JSON.parse(raw);
    } catch {
      return new NextResponse(raw, { status: upstream.status, headers: outHeaders });
    }
    const res = NextResponse.json(stripClaimToken(body), { status: upstream.status, headers: outHeaders });
    const token = tokenFromClaimBody(body);
    if (upstream.ok && token) {
      res.cookies.set(sessionCookieName, token, sessionCookieAttrs);
    } else {
      const cookies = typeof upstream.headers.getSetCookie === "function" ? upstream.headers.getSetCookie() : [];
      for (const c of cookies) res.headers.append("Set-Cookie", c);
    }
    return res;
  }

  const res = new NextResponse(upstream.body, { status: upstream.status, headers: outHeaders });
  const cookies = typeof upstream.headers.getSetCookie === "function" ? upstream.headers.getSetCookie() : [];
  for (const c of cookies) res.headers.append("Set-Cookie", c);
  return res;
}
