# Dynamic DNS Updater

This service monitors the public IP of your homelab and updates DNS A records through the Spaceship API whenever the IP changes.

## Configuration

Create a `.env` file or set environment variables:

```
SPACESHIP_API_KEY=your-key
SPACESHIP_API_SECRET=your-secret
SPACESHIP_BASE_URL=https://spaceship.dev/api/v1
POLL_INTERVAL_HOURS=24
CACHE_PATH=state/last_ip
IP_ENDPOINTS=https://api.ipify.org,https://ifconfig.me
DRY_RUN=false
```

- `SPACESHIP_API_KEY` / `SPACESHIP_API_SECRET`: API credentials provided by Spaceship.
- `SPACESHIP_BASE_URL`: Override if Spaceship exposes a different API root.
- `POLL_INTERVAL_HOURS`: How often to re-check your external IP (defaults to 24h).
- `CACHE_PATH`: File storing last known IP for change detection.
- `IP_ENDPOINTS`: Optional comma-separated list of services to query for your public IP.
- `DRY_RUN`: Set to `true` to log intended updates without performing them.

The service fetches all domains and DNS records during startup and caches them in memory; when the IP changes, it rewrites every A record to the new address.

## Running

```
go build ./cmd/dnsupdater
./dnsupdater
```

For continuous operation on Proxmox, package the binary in a lightweight container or run it with a simple `systemd` unit:

```
# /etc/systemd/system/dnsupdater.service
[Unit]
Description=Spaceship Dynamic DNS Updater
After=network.target

[Service]
WorkingDirectory=/opt/dnsupdater
ExecStart=/opt/dnsupdater/dnsupdater
EnvironmentFile=/opt/dnsupdater/.env
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Pair it with a timer if you prefer scheduled invocations instead of a long-running process. Logs are emitted to stdout for easy collection.

## Development

- `go test ./...`
- Default IP polling sources: `api.ipify.org`, `ifconfig.me`, `checkip.amazonaws.com`.

