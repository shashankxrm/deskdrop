"use client";

import { useCallback, useEffect, useState } from "react";
import { DeviceList } from "@/components/DeviceList";
import { InviteQR } from "@/components/InviteQR";
import { PairPanel } from "@/components/PairPanel";
import {
  claimWeb,
  clearSession,
  createPairing,
  deleteAccount,
  listDevices,
  listHistory,
  revokeDevice,
  type Device,
  type HistoryLink,
} from "@/lib/api";

export default function HomePage() {
  const [phase, setPhase] = useState<"loading" | "pair" | "app">("loading");
  const [devices, setDevices] = useState<Device[]>([]);
  const [links, setLinks] = useState<HistoryLink[]>([]);
  const [invite, setInvite] = useState<{ code: string; expires_at: string } | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [busyId, setBusyId] = useState("");
  const [confirm, setConfirm] = useState("");

  const refresh = useCallback(async () => {
    const d = await listDevices();
    if (!d.ok) {
      if (d.status === 401) await clearSession();
      setPhase("pair");
      setDevices([]);
      return;
    }
    setDevices(d.devices);
    setPhase("app");
    try {
      setLinks(await listHistory());
    } catch {
      setLinks([]);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function startNew() {
    setBusy(true);
    setError("");
    try {
      await clearSession();
      const first = await createPairing();
      await claimWeb(first.code, "Dashboard");
      const extra = await createPairing();
      setInvite(extra);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  }

  async function join(code: string) {
    setBusy(true);
    setError("");
    try {
      await claimWeb(code, "Dashboard");
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  }

  async function mintInvite() {
    setBusy(true);
    setError("");
    try {
      setInvite(await createPairing());
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  }

  async function onRevoke(id: string) {
    setBusyId(id);
    setError("");
    try {
      await revokeDevice(id);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed");
    } finally {
      setBusyId("");
    }
  }

  async function onDelete() {
    if (confirm !== "DELETE") return;
    setBusy(true);
    try {
      await deleteAccount();
      setInvite(null);
      setConfirm("");
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    await fetch("/logout", { method: "POST", credentials: "include" });
    setInvite(null);
    await refresh();
  }

  if (phase === "loading") return <p>Loading…</p>;
  if (phase === "pair") {
    return <PairPanel busy={busy} error={error} onStart={() => void startNew()} onJoin={(c) => void join(c)} />;
  }

  return (
    <main>
      <h1>deskdrop</h1>
      {error ? <p role="alert">{error}</p> : null}
      <p className="row">
        <button type="button" onClick={() => void mintInvite()} disabled={busy}>
          Add a phone or laptop
        </button>
        <button type="button" onClick={() => void logout()}>
          Sign out
        </button>
      </p>
      {invite ? (
        <InviteQR
          code={invite.code}
          origin={typeof window !== "undefined" ? window.location.origin : ""}
          expiresAt={invite.expires_at}
        />
      ) : (
        <p>Tap <strong>Add a phone or laptop</strong> for a QR (phone + Mac). Pair the laptop before you share.</p>
      )}
      <h2>Devices</h2>
      <DeviceList devices={devices} onRevoke={(id) => void onRevoke(id)} busyId={busyId} />
      <h2>History</h2>
      {links.length === 0 ? (
        <p>No links yet.</p>
      ) : (
        <ul>
          {links.map((l) => (
            <li key={l.id}>
              <a href={l.url} target="_blank" rel="noreferrer">
                {l.title || l.url}
              </a>
            </li>
          ))}
        </ul>
      )}
      <h2>Delete account</h2>
      <p>Wipes this pairing chain on the server. Type DELETE to confirm.</p>
      <input aria-label="Type DELETE to confirm" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
      <button type="button" disabled={busy || confirm !== "DELETE"} onClick={() => void onDelete()}>
        Delete account
      </button>
    </main>
  );
}
