"use client";

import { useEffect, useState } from "react";
import { qrPayload } from "@/lib/pairing";

type Props = {
  code: string;
  expiresAt?: string;
};

export function InviteQR({ code, expiresAt }: Props) {
  const [src, setSrc] = useState("");
  useEffect(() => {
    let cancelled = false;
    import("qrcode").then((QR) =>
      QR.toDataURL(qrPayload(code), { margin: 1, width: 220 }).then((url) => {
        if (!cancelled) setSrc(url);
      }),
    );
    return () => {
      cancelled = true;
    };
  }, [code]);
  return (
    <figure>
      {src ? <img src={src} width={220} height={220} alt={`QR for ${code}`} /> : <p>QR…</p>}
      <figcaption>
        <code data-testid="invite-code">{code}</code>
        {expiresAt ? <span> expires {expiresAt}</span> : null}
        <p>Phone: scan or paste. Laptop: <code>deskdrop-agent pair -code {code}</code></p>
      </figcaption>
    </figure>
  );
}
