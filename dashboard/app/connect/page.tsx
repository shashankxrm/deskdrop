"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";

function CodeFromQR() {
  const code = useSearchParams().get("code");
  if (!code) return null;
  return (
    <p>
      Pairing code from this QR: <code>{code}</code>. Open deskdrop on the phone and Scan this same QR (or paste the
      code).
    </p>
  );
}

export default function ConnectPage() {
  return (
    <main>
      <h1>Connect phone and laptop</h1>
      <Suspense>
        <CodeFromQR />
      </Suspense>
      <ol>
        <li>
          On this computer, go to the{" "}
          <a href="/">home page</a> and tap <strong>Start a new pairing</strong> (or{" "}
          <strong>Add a phone or laptop</strong> if you are already signed in). A QR appears.
        </li>
        <li>
          On the phone, open deskdrop. <strong>Scan QR</strong> then <strong>Pair as phone</strong>. The QR is a link to
          this site plus the code — it fills API URL and code. You do not need Forget first.
        </li>
        <li>
          On the Mac, with the same code:
          <pre>
            deskdrop-agent pair -code ABCD-EFGH{"\n"}
            deskdrop-agent run
          </pre>
        </li>
        <li>Share a link from the phone. The laptop browser opens it within about 10 seconds.</li>
      </ol>
      <p>
        Pair the laptop <strong>before</strong> you share. A new “Start a new pairing” is a new user — use Add a
        phone or laptop instead.
      </p>
      <p>
        <a href="/">Back</a>
      </p>
    </main>
  );
}
