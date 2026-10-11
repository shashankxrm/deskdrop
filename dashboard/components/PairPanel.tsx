import { useState } from "react";
import { codeComplete, normalizeCode } from "@/lib/pairing";

type Props = {
  busy?: boolean;
  error?: string;
  onStart: () => void;
  onJoin: (code: string) => void;
};

export function PairPanel({ busy, error, onStart, onJoin }: Props) {
  const [code, setCode] = useState("");
  const ready = codeComplete(code);
  return (
    <section>
      <h1>deskdrop</h1>
      <p>
        Start here on the computer. Then scan the QR with the phone (it fills the public HTTPS URL). Then pair the
        Mac with the same code. <a href="/connect">Full steps</a>.
      </p>
      {error ? <p role="alert">{error}</p> : null}
      <p>
        <button type="button" disabled={busy} onClick={onStart}>
          Start a new pairing
        </button>
      </p>
      <p>Or paste a code from a phone or laptop already on the chain:</p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (ready) onJoin(normalizeCode(code));
        }}
      >
        <label>
          Pairing code
          <input
            aria-label="Pairing code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            autoComplete="off"
            spellCheck={false}
          />
        </label>
        <button type="submit" disabled={busy || !ready}>
          Join
        </button>
      </form>
    </section>
  );
}
