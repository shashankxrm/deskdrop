# Dashboard (Next.js)

C4 is a Next.js app in `dashboard/`. It pairs as a **web** device, shows a QR for extra phone/laptop codes, lists devices and history, and can delete the account. The browser never stores the raw device token: claim JSON is stripped by the same-origin proxy; the API sets httpOnly `deskdrop_device`. See [ADR 0003](../adr/0003-cookie-vs-bearer.md) and [privacy.md](../privacy.md).

Locally the dashboard is `http://127.0.0.1:3000` and proxies `/v1` to the Compose API. Production is same-origin at https://shashank-dropdesk.duckdns.org. How to pair: [connect.md](../connect.md) and `/connect`.

## Why this way

- Start a new pairing: bootstrap code, immediately claim it as `web`, then mint a **second** code + QR. Phone and Mac both claim that same QR until it expires (~3 minutes). The first code is already spent by the dashboard.
- Join: paste a code minted by an already-paired device.
- Unpair (`DELETE /v1/devices/{id}`) on a laptop row: the agent’s next poll is 401.
- Delete account wipes the chain and clears the cookie.

## Failure modes

- API down: pair/start errors; Playwright fails `/readyz`.
- Expired/used code: 410/409 shown as `error`.
- Revoked cookie: device list 401 → pair screen.
- Token not in localStorage / claim JSON (proxy strips it).

## Verify or demo

Compose API must be up.

```bash
cd dashboard
npm install
npx playwright install chromium
npm test
# Vitest: pairing normalize + PairPanel + DeviceList

npm run test:e2e
# Playwright: Start a new pairing → Devices heading, type web, invite QR

npm run dev
# open http://127.0.0.1:3000
```

**Start a new pairing** is a new user (new QR for the phone). To attach this browser to the Mac+phone you already have:

```bash
go build -o /tmp/deskdrop-agent ./cmd/agent
/tmp/deskdrop-agent invite
# paste that code into Join on http://127.0.0.1:3000
```

Unpair the laptop row, then in the agent log the next poll is `http 401`. Delete account → type `DELETE` → pair screen again.

GitHub Actions Vitest/Playwright is D1.
