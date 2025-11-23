# Dynamic DNS Updater

This service monitors the public IP of your homelab and updates DNS A records through the Spaceship API whenever the IP changes.

## Configuration

Create a `.env` file or set environment variables:

```
SPACESHIP_API_KEY=your-key
SPACESHIP_API_SECRET=your-secret
SPACESHIP_BASE_URL=https://spaceship.dev/api/v1
POLL_INTERVAL_HOURS=24
IP_ENDPOINTS=https://api.ipify.org,https://ifconfig.me
DRY_RUN=false
```

- `SPACESHIP_API_KEY` / `SPACESHIP_API_SECRET`: API credentials provided by Spaceship.
- `SPACESHIP_BASE_URL`: Override if Spaceship exposes a different API root.
- `POLL_INTERVAL_HOURS`: How often to re-check your external IP (defaults to 24h).
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

## Docker

### Building Locally

Build the Docker image:

```bash
docker build -t dnsupdater:latest .
```

Run the container:

```bash
docker run -d \
  --name dnsupdater \
  --restart unless-stopped \
  --env-file .env \
  dnsupdater:latest
```

### Using Pre-built Images

Pre-built images are available on GitHub Container Registry. Pull the latest image:

```bash
docker pull ghcr.io/erkkip/spaceship-dns-sync:latest
```

### Docker Compose

For easier deployment, use the provided `docker-compose.example.yml`:

1. Copy the example file:
   ```bash
   cp docker-compose.example.yml docker-compose.yml
   ```

2. Ensure your `.env` file is in the same directory with the required configuration.

3. Start the service:
   ```bash
   docker-compose up -d
   ```

The last known IP address is stored in memory and will be checked on startup. No volume mounts are required.

## Development

- `go test ./...`
- Default IP polling sources: `api.ipify.org`, `ifconfig.me`, `checkip.amazonaws.com`.

