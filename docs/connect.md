# Connect phone, laptop, and this site

Public beta: **https://shashank-dropdesk.duckdns.org** (start here; README install: [../README.md](../README.md)).

You are pairing three things onto **one chain**. A share only opens on laptops that were already paired **before** you sent the link.

## Happy path (about one minute)

1. **Computer browser** — open https://shashank-dropdesk.duckdns.org  
   Tap **Start a new pairing**. A QR appears. It is a normal HTTPS link (`/connect?code=ABCD-EFGH`) so the phone scanner keeps both the site and the code. Codes last ~3 minutes; tap **Add a phone or laptop** for a fresh one.

2. **Phone** — open deskdrop (rebuild if Scan still dumps the host into the code box).  
   **Scan QR**, then **Pair as phone**. The scan should set API URL to `https://shashank-dropdesk.duckdns.org` and the eight-character code.  
   If you were paired to localhost before, Scan still works: it replaces the old pairing. You do not have to tap Forget first.

3. **Mac** — same QR/code:

   ```bash
   deskdrop-agent pair -code ABCD-EFGH
   deskdrop-agent run
   ```

   Default API is the public HTTPS host. Then `deskdrop-agent install` if you want it at login.

4. **Share** a YouTube (or other) URL to deskdrop. A tab should open on the Mac within ~10s. The same link shows under **History** on the site.

## If something is missing

| You see | Cause | Fix |
|---|---|---|
| `http 409: already used` | That code was already claimed (phone scan spends it on old API) or 8 devices on the chain | **Add a phone or laptop** for a fresh code. `run` does nothing until `pair` prints `paired device_id=…` |
| Phone pairs, Mac never opens the link | Laptop was not on this chain yet, or agent is still on `127.0.0.1` | Pair the agent with the dashboard QR, `run`, **share again** |
| Scan does nothing useful | You only scanned and did not tap Pair | Tap **Pair as phone** (or Scan again — it auto-pairs when the QR has the site URL) |
| Dashboard History empty | Phone is on a different chain than the site | Start on the **website**, then Pair the phone to **that** QR |
| `unauthorized` on the site | Stale cookie | Refresh, **Start a new pairing** again |
| Gatekeeper blocks the agent | Unsigned build | System Settings → Privacy & Security → Open Anyway |

## What not to do

- Do not type a pairing **code** into `PHONE=` or Bearer. The code is eight characters (`ABCD-EFGH`). The phone token is a long hex string and stays on the device.
- Do not leave the phone API URL as `http://127.0.0.1:8080` or a LAN IP once you are on DuckDNS.
- Do not start a **second** “Start a new pairing” for the laptop — that is a new user. Use **Add a phone or laptop** on the same dashboard session.

Local Compose (no public HTTPS): site `http://127.0.0.1:3000`, phone API `http://<your-lan-ip>:8080`, agent `-api http://127.0.0.1:8080`. Same four steps.
