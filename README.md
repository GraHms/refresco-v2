# refres.co v2

Fan support platform for Mozambican creators. Fans pay with M-Pesa, e-Mola or card on a single public page; supports accumulate as balances (split across group members); creators request withdrawals that the super admin approves and pays within 72 h. Sign-up is invite only.

```
backend/   Go 1.26 · Gin · GORM (Postgres, automigrate) · ports and adapters
frontend/  React 18 · Vite · TypeScript (SPA, served by nginx)
```

## Quick start (local)

```bash
make dev-db          # Postgres 16 in Docker on :5432 (or use your own)
make seed            # demo data: Tu Pra Tu Podcast, invites, 60 days of supports
make api             # API on :8080
make web             # SPA on :5173, proxies /api to :8080
```

Demo logins after `make seed`: creator `cardo@tupratu.co.mz` / `tupratu-2026`; super admin `admin@refres.co` / `admin-refres-2026`. Public page: `/tupratu`. Invite codes: `REFRES-BETA`, `MAPUTO-01`.

## Production

```bash
cp .env.example .env   # fill in every secret (openssl rand -base64 48)
make docker-build
make up                # db + api + web; web listens on $WEB_PORT (default 8080)
```

Put a TLS terminating load balancer in front of `web`. The API refuses to start in `APP_ENV=production` without strong secrets and `COOKIE_SECURE=true`. The super admin is created on first boot from `ADMIN_EMAIL` / `ADMIN_PASSWORD`; remove the password from the environment afterwards.

`make help` lists all targets. `make test` runs `go vet`, the race-enabled Go tests and the TypeScript check. Set `TEST_DATABASE_URL` to also run the Postgres integration test (each run uses a throwaway schema).

## Architecture

| Layer | Package | Responsibility |
| --- | --- | --- |
| Domain | `internal/domain` | Entities, money in centavos, fee in basis points, split rounding, validation |
| Ports | `internal/ports` | Repository, metrics, payment gateway, notifier, hasher, token and clock interfaces |
| Application | `internal/app` | Use cases: auth and invites, donations and settlement, backoffice, group split approval, wallet and withdrawals, admin |
| Adapters | `internal/adapters/*` | `postgres` (GORM models, repos, aggregate metrics), `payments/mock`, `http` (Gin), `security` (bcrypt, JWT), `notify` |

**Money flow.** A confirmed support is settled once, inside a transaction that locks the support and member rows: fee = effective account fee (negotiated fees can expire), net split by member percentages with the rounding remainder to the admin, balances credited, a ledger entry per credit, and the daily aggregate updated. Withdrawal requests debit the balance immediately (hold); rejection refunds it with a ledger entry.

**Metrics are aggregates.** Every dashboard figure reads from `daily_stats` (one row per account, local day and channel, upserted on settlement) or from SQL aggregates (`SUM`, `COUNT DISTINCT`, `mode()`), never by loading and counting rows in Go.

**Payments.** `PAYMENTS_PROVIDER=mock` is the only gateway for now. Wallet requests confirm after `MOCK_AUTO_CONFIRM`; numbers ending in `0000` fail (insufficient funds), `9999` never confirm. Card OTP `000000` is rejected. The SPA never sends card numbers to the API, only brand, last four digits and a gateway token. A signed webhook endpoint (`POST /api/v1/webhooks/payments`, HMAC-SHA256 in `X-Mock-Signature`) is in place for real providers. `PAYMENTS_ALLOW_SIMULATE` (off in production) enables the "tap the phone to confirm" sandbox shortcut.

## Security

- Session JWT in an `HttpOnly`, `Secure`, `SameSite=Strict` cookie scoped to `/api`; token version is bumped on password reset, revoking old sessions.
- Mutating requests must be JSON and, when `ALLOWED_ORIGINS` is set, come from an allowed origin.
- bcrypt (cost 12) with constant-time behaviour for unknown users; reset codes are HMAC'd, expire in 10 minutes, allow 5 attempts and are exchanged for a single-use reset token.
- Per-IP rate limits on auth, payment, polling and the API in general; 64 KB request body limit; strict timeouts.
- Row-level locking for invites, balances, supports and withdrawals; DB check constraints keep balances non-negative and invite uses within limits.
- CSV export neutralises formula injection. nginx sets CSP, HSTS, frame, referrer and permissions headers. Containers run as non-root with a read-only filesystem and no capabilities; Postgres is on an internal network.

## API

All routes are under `/api/v1`. Public: `pages/:handle`, `pages/:handle/supports`, `supports/:id` (+ `/otp`, `/resend`, `/cancel`), `auth/*`, `handles/:handle`, `split-approvals/:token`. Creator: `backoffice/*` (overview, supports + CSV, page, group, wallet). Super admin: `admin/*` (dashboard, accounts, fee, status, withdrawals with detail and PDF payment orders at `withdrawals/:id/pdf` and `withdrawals.pdf?status=approved`, invites). Amounts are integers in centavos.
