# DietDaemon CI/CD — Runner & Deploy Setup

## Self-Hosted GitHub Actions Runner

### Security: Runner Scope

**Self-hosted runner is ONLY used for trusted events** — pushes to `main` and
tagged releases (`v*`). These require write access to the repo.

**PR checks run on GitHub-hosted runners** (`ubuntu-latest`). Fork PRs never
touch the homelab. This prevents arbitrary code execution from untrusted
contributors.

Runner is still needed for:
- `main.yml` → `docker` job (BuildKit + GHCR push)
- `main.yml` → `deploy` job (Watchtower notification)
- `release.yml` → `docker` job (versioned image push)

If you skip Docker builds entirely (e.g., use GitHub-hosted + Docker),
you don't need a self-hosted runner at all.

### Target Machine

**Ryzen 7 7840HS** (8c/16t, 32 GB RAM, NVMe SSD).

The i5-10th-gen server works as a backup. Register only one runner per repo to avoid
duplicate job dispatch. If you need failover, add the second machine with a different
label (e.g. `homelab-backup`) and update the workflow `runs-on` arrays.

### Prerequisites

```bash
# Verify versions on the runner machine
go version          # go1.26.x
node --version      # v22.x
npm --version       # 11.x
docker --version    # 27.x+
docker buildx version  # BuildKit present

# Install buildx if missing
docker buildx install
```

### Install the Runner

```bash
# 1. Create a dedicated user
sudo useradd -r -m -s /usr/bin/bash gh-runner

# 2. Create the runner directory
sudo -u gh-runner mkdir -p /home/gh-runner/actions-runner
cd /home/gh-runner/actions-runner

# 3. Download the runner package (check for the latest version)
curl -o actions-runner-linux-x64.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.322.0/actions-runner-linux-x64-2.322.0.tar.gz
tar xzf actions-runner-linux-x64.tar.gz

# 4. Configure
#    Get the token from: https://github.com/gsaraiva2109/DietDaemon/settings/actions/runners/new
sudo -u gh-runner ./config.sh \
  --url https://github.com/gsaraiva2109/DietDaemon \
  --token <REGISTRATION_TOKEN> \
  --labels self-hosted,homelab,linux,amd64 \
  --name dietdaemon-runner \
  --work _work

# 5. Install and start as a systemd service
sudo ./svc.sh install gh-runner
sudo ./svc.sh start

# 6. Verify
sudo ./svc.sh status
# Expected: "active (running)"
```

### Docker Access for Runner

The runner needs Docker access to build and push images.

```bash
sudo usermod -aG docker gh-runner
# Log out / log in for group membership to take effect.
# Or restart the runner service:
sudo ./svc.sh stop && sudo ./svc.sh start
```

**No Docker-in-Docker needed.** The runner uses the host Docker daemon directly
(same as `docker build` from your shell). This is safe for a single-tenant homelab.

### Verify in GitHub

Go to **Repo → Settings → Actions → Runners**. You should see `dietdaemon-runner`
as **Idle**. The labels `self-hosted`, `homelab`, `linux`, `amd64` must appear.

---

## Deployment: Watchtower (Recommended)

Watchtower monitors `ghcr.io/gsaraiva2109/dietdaemon:latest` and auto-updates the
running container when it detects a new image.

### How It Works

```
CI (main.yml)                     Homelab
     │                                │
     ├─ docker build + push ─────────►│ ghcr.io/.../dietdaemon:latest
     │                                │
     │                    ┌───────────▼───────────┐
     │                    │  Watchtower (poll 60s) │
     │                    │  detects new :latest    │
     │                    │  docker compose pull    │
     │                    │  docker compose up -d   │
     │                    └───────────────────────┘
```

### Enable

Watchtower is already configured in `docker-compose.yml`. Start:

```bash
docker compose up -d
```

Both `dietdaemon` and `watchtower` start. Watchtower begins polling immediately.

### Verify

```bash
# Check Watchtower logs
docker compose logs watchtower

# Expected output (on first run):
# "Watchtower X.Y.Z — Monitoring dietdaemon"
```

After CI pushes a new image, Watchtower logs show:

```
"Found new ghcr.io/gsaraiva2109/dietdaemon:latest image"
"Stopping dietdaemon ..."
"Starting dietdaemon ..."
```

### Manual Pull (Bypass Watchtower)

```bash
docker compose pull dietdaemon
docker compose up -d --no-deps dietdaemon
```

---

## Deployment: Webhook (Alternative)

Faster deploys than Watchtower's 60s poll. CI POSTs to an endpoint on your homelab
after pushing the image.

### Setup (systemd socket activation)

```bash
# 1. Create the webhook handler
sudo cp deploy/webhook.sh /usr/local/bin/dietdaemon-webhook
sudo chmod +x /usr/local/bin/dietdaemon-webhook

# 2. Install systemd units
sudo cp deploy/dietdaemon-webhook.socket /etc/systemd/system/
sudo cp deploy/dietdaemon-webhook@.service /etc/systemd/system/

# 3. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable --now dietdaemon-webhook.socket
```

### Configure CI to Use Webhook

1. Add a repository secret: **Settings → Secrets and variables → Actions**
   - Name: `DEPLOY_WEBHOOK_URL`
   - Value: `http://<homelab-ip>:9876`

2. Update `main.yml` deploy job to POST to the webhook URL (uncomment the curl
   line in the deploy job).

---

## Repository Secrets

Configure these in **GitHub Repo → Settings → Secrets and variables → Actions**:

| Secret | Required | Purpose |
|--------|----------|---------|
| `GITHUB_TOKEN` | Yes | Auto-provided by GitHub. Used by `docker/login-action` to push to GHCR. No manual setup needed. |
| `NTFY_DEPLOY_URL` | No | ntfy topic URL for deploy failure notifications. E.g. `https://ntfy.sh/dietdaemon-deploy`. If not set, failures are silent. |
| `DEPLOY_WEBHOOK_URL` | No | Only needed if using the webhook alternative instead of Watchtower. |

`GITHUB_TOKEN` is a built-in secret — you don't need to create it. It has
`write:packages` scope in workflows that need to push to GHCR.

---

## GHCR Package Visibility

By default, GHCR packages are **private**. To make the image pullable from
your homelab without extra auth:

1. Go to **GitHub → Your profile → Packages → dietdaemon**
2. Click **Package settings**
3. Under **Danger Zone → Change visibility**, set to **Public**

Or keep it private and create a **classic personal access token** with
`read:packages` scope. Store it on the homelab and `docker login ghcr.io`
before pulling.

### Docker Login on Homelab (if private)

```bash
echo "<GITHUB_PAT>" | docker login ghcr.io -u gsaraiva2109 --password-stdin
```

---

## Troubleshooting

### Runner shows "Offline" in GitHub

```bash
ssh <runner-host>
sudo systemctl status actions.runner.*
sudo journalctl -u actions.runner.* -n 50
```

Common causes:
- Token expired (re-register with a new token)
- Network change (runner can't reach github.com)
- Disk full (`/home/gh-runner/actions-runner/_work`)

### Docker build fails with "no space left on device"

```bash
docker system prune -af
# Or clear BuildKit cache specifically:
docker buildx prune -af
```

### Watchtower not pulling

```bash
docker compose logs watchtower
```

Common causes:
- Image is private on GHCR (run `docker login ghcr.io` on the host)
- `WATCHTOWER_POLL_INTERVAL` too high (reduce to 30s)
- Container not labeled (ensure `com.centurylinklabs.watchtower.enable=true`)

### ntfy notifications not arriving

- Verify `NTFY_DEPLOY_URL` secret is set
- Test manually: `curl -H "Title: Test" -H "Priority: high" -d "test" https://ntfy.sh/dietdaemon-deploy`
