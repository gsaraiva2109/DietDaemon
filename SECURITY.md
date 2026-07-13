# Security Policy

## Supported versions

| Version  | Supported          |
|----------|--------------------|
| latest   | :white_check_mark: |
| < latest | :x:                |

Only the latest release receives security patches. DietDaemon is self-hosted —
there is no hosted service to patch, so update your instance promptly.

## Reporting a vulnerability

**Do not open a public issue.** Send details to:

- GitHub: [Report a vulnerability](https://github.com/gsaraiva2109/dietdaemon/security/advisories/new) (private)
- Or email: see [ATTRIBUTION.md](./ATTRIBUTION.md) for contact

Expect:
- **Acknowledgment** within 72 hours
- **Status update** within 7 days
- **Disclosure** after a fix is released and users have had a reasonable window to update

## Scope

Relevant vulnerabilities include (but aren't limited to):

- **Authentication bypass** — OIDC, WebAuthn, TOTP flows
- **Data leakage** — meal data, weight, sleep logs exposed across users
- **Injection** — SQL, command injection via chat input or API
- **SSRF** — via the Ollama sidecar or web fetch tools
- **Configuration exposure** — `.env` secrets leaked via logs, API, or dashboard

## Out of scope

- Issues in third-party services (Telegram, Discord, Matrix) — report to them
- Issues requiring physical access to the host machine
- Denial of service from an authenticated user

## Hall of fame

We'll credit you in the release notes (or keep you anonymous, your choice).
