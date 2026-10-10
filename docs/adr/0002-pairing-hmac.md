# ADR 0002: Bootstrap pairing and HMAC-SHA256 tokens

## Status

Accepted (B2)

## Context

v1 split phone and laptop into different users (compiled-in bearer vs WebAuthn). v2 has no passwords. Device tokens must not sit in Postgres as plaintext.

## Decision

- A user is created by an unauthenticated `POST /v1/pairings` (bootstrap). The response is a short, single-use code (TTL 3 minutes). `POST /v1/pairings/claim` returns a device token **once**.
- Later codes require `Authorization: Bearer` of an already-paired device. JSON `userId` is ignored.
- At rest, pairing codes and device tokens are `HMAC-SHA256(pepper, plaintext)` with `DEVICE_TOKEN_PEPPER`. Lookup is by hash. Not `sha256(pepper || token)`.
- Failed claims are counted per IP in process memory; the 6th failure in a minute is 429.

## Alternatives

- Email / magic link / WebAuthn: passwords-by-another-name and v1’s identity split.
- SHA-256(pepper || token): length-extension and a weaker construction; HMAC is the usual keyed hash.
- Redis for claim rate limits: extra process. Per-pod in-memory is enough at 20 users (global limiter is Post-v2).

## Consequences

- Losing the pepper invalidates every stored hash (tokens still work until restart if we hashed wrong — we must never change the construction without a migration).
- Two API replicas (D4) each have their own claim counter. Documented; Postgres-backed limiter is Post-v2.
