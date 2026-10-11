import { proxyToAPI } from "@/lib/proxy";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

type Ctx = { params: Promise<{ path: string[] }> };

async function handle(req: Request, ctx: Ctx): Promise<Response> {
  const { path } = await ctx.params;
  return proxyToAPI(req, path);
}

export const GET = handle;
export const POST = handle;
export const PATCH = handle;
export const DELETE = handle;
export const OPTIONS = handle;
