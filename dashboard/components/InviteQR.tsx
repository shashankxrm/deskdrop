"use client";

import { useEffect, useState } from "react";
import { qrPayload } from "@/lib/pairing";

type Props = {
  code: string;
  origin?: string;
  expiresAt?: string;
};

export function InviteQR({ code, origin = "", expiresAt }: Props) {
  const [src, setSrc] = useState("");
  const payload = qrPayload(code, origin);
  useEffect(() => {
    let cancelled = false;
    import("qrcode").then((QR) =>
      QR.toDataURL(payload, { margin: 1, width: 220 }).then((url) => {
        if (!cancelled) setSrc(url);
      }),
    );
    return () => {
      cancelled = true;
    };
  }, [payload]);
  return (
    <figure>
      {src ? <img src={src} width={220} height={220} alt={`QR for ${code}`} /> : <p>QR…</p>}
      <figcaption>
        <code data-testid="invite-code">{code}</code>
        {expiresAt ? <span> expires {expiresAt}</span> : null}
        <ol>
          <li>
            Phone: Scan this QR, then <strong>Pair as phone</strong>. It sets API URL to{" "}
            <code>{origin || "this site"}</code> and the code above.
          </li>
          <li>
            Mac: <code>deskdrop-agent pair -code {code}</code> then <code>deskdrop-agent run</code>
          </li>
        </ol>
      </figcaption>
    </figure>
  );
}
