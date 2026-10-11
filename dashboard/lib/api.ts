export type Device = {
  id: string;
  name: string;
  type: string;
  created_at: string;
  last_seen_at?: string;
};

export type HistoryLink = {
  id: string;
  url: string;
  title?: string;
  created_at: string;
};

async function parse(res: Response): Promise<unknown> {
  const text = await res.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return { error: text };
  }
}

function errMsg(body: unknown, fallback: string): string {
  if (body && typeof body === "object" && "error" in body) {
    const e = (body as { error: unknown }).error;
    if (typeof e === "string" && e) return e;
  }
  return fallback;
}

export async function api(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(path, { ...init, headers, credentials: "include" });
}

export async function clearSession(): Promise<void> {
  await fetch("/logout", { method: "POST", credentials: "include" });
}

export async function listDevices(): Promise<{ ok: true; devices: Device[] } | { ok: false; status: number; error: string }> {
  const res = await api("/v1/devices");
  const body = await parse(res);
  if (res.status === 401) return { ok: false, status: 401, error: "unauthorized" };
  if (!res.ok) return { ok: false, status: res.status, error: errMsg(body, "failed") };
  const devices = (body as { devices?: Device[] }).devices ?? [];
  return { ok: true, devices };
}

export async function createPairing(): Promise<{ code: string; expires_at: string }> {
  let res = await api("/v1/pairings", { method: "POST", body: "{}" });
  if (res.status === 401) {
    await clearSession();
    res = await api("/v1/pairings", { method: "POST", body: "{}" });
  }
  const body = await parse(res);
  if (!res.ok) throw new Error(errMsg(body, `pairing create ${res.status}`));
  return body as { code: string; expires_at: string };
}

export async function claimWeb(code: string, name: string): Promise<void> {
  const res = await api("/v1/pairings/claim", {
    method: "POST",
    body: JSON.stringify({ code, type: "web", name }),
  });
  const body = await parse(res);
  if (!res.ok) throw new Error(errMsg(body, `claim ${res.status}`));
}

export async function revokeDevice(id: string): Promise<void> {
  const res = await api(`/v1/devices/${id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) {
    const body = await parse(res);
    throw new Error(errMsg(body, `revoke ${res.status}`));
  }
}

export async function listHistory(): Promise<HistoryLink[]> {
  const res = await api("/v1/links?limit=20");
  const body = await parse(res);
  if (!res.ok) throw new Error(errMsg(body, "history failed"));
  return (body as { links?: HistoryLink[] }).links ?? [];
}

export async function deleteAccount(): Promise<void> {
  const res = await api("/v1/account", { method: "DELETE" });
  if (!res.ok && res.status !== 204) {
    const body = await parse(res);
    throw new Error(errMsg(body, `delete ${res.status}`));
  }
}
