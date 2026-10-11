import { NextResponse } from "next/server";
import { sessionCookieName } from "@/lib/session";

export function POST() {
  const res = NextResponse.json({ ok: true });
  res.cookies.set(sessionCookieName, "", {
    path: "/",
    maxAge: 0,
    httpOnly: true,
    sameSite: "lax",
  });
  return res;
}
