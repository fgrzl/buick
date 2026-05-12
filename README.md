# buick

Small HTTP/HTTPS reverse proxy for local development. It routes by `Host`, forwards with `httputil.ReverseProxy`, supports WebSocket upgrades, and reads a YAML service table.

Two binaries:

- **`buickd`** — HTTP/HTTPS reverse proxy (runs until stopped).
- **`buick`** — developer CLI: validate config, print routes, TLS/CA setup (`buick init`), and related helpers.

## Install the `buick` CLI

You need [Go](https://go.dev/dl/) **1.22 or newer** (matches this module’s `go` directive).

**Install a released build from the module path** (binary name: `buick`):

```bash
go install github.com/fgrzl/buick/cmd/buick@latest
```

Put Go’s install directory on your `PATH` if it is not already (commonly **`$HOME/go/bin`** on Linux and macOS, **`%USERPROFILE%\go\bin`** on Windows). Then confirm:

```bash
buick --version
```

**Install from a clone of this repository** (builds whatever revision you have checked out):

```bash
go install ./cmd/buick
```

**Build without installing** (writes `buick` in the current directory):

```bash
go build -o buick ./cmd/buick
```

**Private forks or vanity imports:** If `go install` cannot fetch the module, configure [`GOPRIVATE`](https://go.dev/ref/mod#private-modules) and Git/SSH access for your host, or use `go install` from a local clone as above.

## Usage

Build the daemon from a clone:

```bash
go build -o buickd ./cmd/buickd
```

### Config files in this repo

| File | Role |
|------|------|
| **`compose.yml`** | Local integration stack: **nginx** backends `service1`–`service3`, **buickd** on network `buick-integration`, host ports **18080** (HTTP) / **18443** (HTTPS). |
| **`compose.buick.yml`** | Config **inside** that stack: listens **`:8080` / `:8443`**, certs under **`/etc/buick/certs/`**, upstreams **`http://service1:8080`** (Compose DNS names). |
| **`buick.yml`** | Same upstream style as Compose (`http://serviceN:8080`) for **in-network** routing; **`./certs/`** paths suit bind-mounting or generating PEMs on the host. |
| **`buick.host.example.yml`** | Example when **buickd runs on the host** and upstreams are on **`127.0.0.1`** ports. Copy or merge into your own file. |

### Run the sample stack (Docker)

From the repository root (builds **buickd** image, starts nginx + proxy):

```bash
docker compose up -d --build
```

Then open **http://127.0.0.1:18080/** with `Host: service1.localhost` (or add `127.0.0.1 service1.localhost` to your hosts file and use the hostname in the browser).

### Run buickd on the host

Use **`buick.host.example.yml`** (or equivalent `target: "http://127.0.0.1:…"`) so upstreams resolve. Running **`./buickd --config ./buick.yml`** on the laptop **without** Docker DNS for `service1` will not reach the sample upstreams.

```bash
./buickd --config ./buick.host.example.yml
```

Validate or print routes (point `--config` at the file you actually use):

```bash
buick --check --config ./buick.host.example.yml
buick --print-routes --config ./compose.buick.yml
```

**Prepare TLS and trust on your machine** (paths and SANs follow the YAML you pass in):

```bash
buick init --config ./buick.host.example.yml
```

See `buick init -h` for flags such as `--skip-trust` (PEM files only) and `--uninstall`.

## Configuration

Top-level fields:

- `proxy.http`: listen address for HTTP. Default in examples is **`:80`**. Omit to disable.
- `proxy.https`: listen address for HTTPS. Default in examples is **`:443`**. Omit to disable.
- `proxy.cert_file` / `proxy.key_file`: PEM paths used by the HTTPS listener. If **both** files exist they are reused; if **either** is missing, **buickd** generates a self-signed pair (see TLS below).

Each `services` entry maps a hostname to an upstream:

- `target`: absolute `http://` or `https://` URL for the upstream. The hostname in the URL is whatever **buickd** must dial: use **Compose service names** (for example **`http://api:8080`**) when **buickd** shares a user-defined network with the backend; use **`http://host.docker.internal:PORT`** when the backend listens on the **host** and **buickd** runs in a container; use **`http://127.0.0.1:PORT`** when **buickd** runs on the **same machine** as the backend.
- `targets`: optional list of absolute upstream URLs for the same hostname. When set (non-empty), **buickd** round-robins across them; do not set `target` in the same entry. There is **no** health checking: every peer is chosen in rotation regardless of availability (use an external load balancer or orchestrator health if you need that).
- `websocket`: when true, disables response buffering (`FlushInterval`) suitable for upgrades.
- `read_timeout` / `write_timeout`: optional `time.ParseDuration` strings. Defaults are 60s for normal HTTP and 168h for websocket services unless overridden.

Incoming requests are matched using `NormalizeHost(r.Host)` so `service1.localhost:80` matches `service1.localhost`. Unknown hosts receive `502 Bad Gateway`.

Forwarded headers set on the upstream request:

- `X-Forwarded-Host` (client `Host`, including port if present)
- `X-Forwarded-Proto` (`http` or `https`)
- `X-Forwarded-For` (appended client IP)

The original client `Host` header is preserved on the proxied request (not rewritten to the upstream hostname).

## Operations

**Management and metrics (loopback only):** On HTTP and HTTPS listeners, **`GET /_buick/health`**, **`GET /_buick/routes`**, and **`GET /_buick/metrics`** are answered only when the **TCP client address is loopback** (`127.0.0.1`, `::1`, etc.), based on `RemoteAddr`. They are intended for local checks and debugging. If **buickd** sits behind another reverse proxy, those paths usually **will not** trigger unless the proxy forwards from loopback or you query **buickd** directly on loopback. **`/_buick/routes`** returns routes **sorted by hostname** so output is stable for scripts and diffs. Unknown paths under **`/_buick/`** (for example a typo) return **404** on loopback.

**Reload:** Send **`SIGHUP`** to reload the route table from the same **`--config`** file (Unix convention; may not apply on all Windows setups). If load or validation fails, the previous routes stay in place. **Listener addresses** and the **HTTP server read/write timeouts** are **not** changed until you restart the process; TLS leaf material may be refreshed via the same dev-cert logic as startup when HTTPS is enabled.

## TLS for local HTTPS

**Recommended on the host:** run **`buick init --config …`** once. It creates a small local CA, issues a leaf certificate for the hostnames in your YAML, writes `proxy.cert_file` / `proxy.key_file`, and installs the CA using **OS tools** (no extra Go modules): **Windows** (`certutil` current-user root store), **macOS** (`security` + login keychain), **Debian-like Linux** (`sudo cp` into `/usr/local/share/ca-certificates` and `update-ca-certificates`). Other platforms, or when install fails, use **`--skip-trust`** and import the generated **`buick-root-ca.pem`** manually. **Firefox NSS** is not modified; Chromium-based browsers on Windows/macOS typically use the OS store.

When `proxy.https` is set, **buickd** still ensures PEM material exists at `cert_file` and `key_file`:

- If **both** files already exist, they are used as-is (including files produced by **`buick init`** or **mkcert**).
- If **either** file is missing, **buickd** generates a **self-signed** RSA certificate valid one year, with SANs for `localhost`, `127.0.0.1`, `::1`, and every hostname key under `services`.

If you skip **`buick init`**, the OS or browser will warn until you trust that certificate (manual import) or replace the PEMs (for example with **mkcert** as below).

## Docker Compose recipe

This repository **does** ship **`compose.yml`** + **`compose.buick.yml`** for the integration stack (see [Integration tests](#integration-tests-docker)). For **your** application, copy or adapt the patterns below into your own Compose file.

### 1. Example `docker-compose.yml` (buickd + persisted certs)

Build context should be the directory that contains this repo (or your own image name). Bind-mount **`./certs`** so TLS files stay on the host across container recreations. Use **`host-gateway`** so `host.docker.internal` and `*.localhost` resolve to the machine running your upstream processes.

The image listens on **80** and **443** by default in the example config. Ports below 1024 often need extra privileges in containers (for example `user: "0:0"` for local dev, or `cap_add: [NET_BIND_SERVICE]` where your runtime supports it). Otherwise, point `proxy.http` / `proxy.https` at high ports (for example `:8080` / `:8443`) and map those in `ports:` instead.

```yaml
services:
  buickd:
    build:
      context: .
      dockerfile: cmd/buickd/Dockerfile
    restart: unless-stopped
    command: ["--config", "/etc/buick/buick.yml"]
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./buick.docker.yml:/etc/buick/buick.yml:ro
      - ./certs:/etc/buick/certs
    extra_hosts:
      - "host.docker.internal:host-gateway"
      - "service1.localhost:host-gateway"
      - "service2.localhost:host-gateway"
      # add one extra_hosts line per hostname you route in buick.yml
    # If buickd cannot write TLS files into ./certs (Linux bind-mount ownership), use:
    # user: "0:0"
```

### 2. Example `buick.docker.yml` (paths and upstreams inside Docker)

Use **absolute paths** under the cert volume and **`http://host.docker.internal:PORT`** for APIs listening on the host (`127.0.0.1` inside the container is not your laptop).

```yaml
proxy:
  http: ":80"
  https: ":443"
  cert_file: "/etc/buick/certs/localhost.pem"
  key_file: "/etc/buick/certs/localhost-key.pem"

services:
  service1.localhost:
    target: "http://host.docker.internal:8080"
  service2.localhost:
    target: "http://host.docker.internal:8081"
```

Upstream ports here are **8080**, **8081**, and so on for each app on the host. **buickd** listens on **80** / **443** in this example, so there is no port clash with those upstreams on the same host.

If **buickd** runs in the **same** Compose project as your APIs, put every service on one user-defined network and use **`http://service1:8080`** style targets instead of `host.docker.internal` (same pattern as **`buick.yml`** / **`compose.buick.yml`** in this repo).

### 3. First run and trust (low friction)

```bash
mkdir -p certs
```

On the **host** (with the **`buick`** CLI [installed](#install-the-buick-cli)), point at the same config paths your Compose volume will use, then bring the stack up:

```bash
buick init --config ./buick.docker.yml
docker compose up -d --build
```

If you prefer not to use **`buick init`**, use one of the following.

**Trust once (pick one):**

1. **buickd-generated self-signed** — After the first successful start, import **`./certs/localhost.pem`** into your OS trust store (or Firefox’s Authorities). **buickd** reuses both PEMs when they already exist, so you do not repeat this unless you delete `./certs`.

2. **mkcert (fewest browser warnings)** — On the host: `mkcert -install` once, then create leaf files in `./certs` with the same names as `cert_file` / `key_file` in your buick config, for example:

```bash
mkcert -cert-file ./certs/localhost.pem -key-file ./certs/localhost-key.pem \
  localhost 127.0.0.1 ::1 service1.localhost service2.localhost service3.localhost
```

Then start Compose. **buickd** will **not** overwrite existing PEMs; trust comes from mkcert’s local CA.

## Integration tests (Docker)

The repo includes **`compose.yml`** with three sample **nginx** backends (`service1`–`service3`, configs under **`tests/integration/nginx/`**) and a **buickd** service built from **`cmd/buickd/Dockerfile`**. **buickd** reads **`compose.buick.yml`** (HTTP **8080**, HTTPS **8443** inside the stack; published as **18080** / **18443** on the host). **buickd** runs as **`user: "0:0"`** in that file so it can write generated TLS material into the named **`buick_certs`** volume (distroless’s default nonroot user cannot create files there on a fresh volume).

```bash
docker compose up -d --build
BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
docker compose down
```

With **`BUICK_INTEGRATION=1`**, tests wait (up to **90s**) for **http://127.0.0.1:18080** to serve the stack so CI is not flaky right after `docker compose up`.

Override the default probe URLs if needed: **`BUICK_HTTP_ADDR`**, **`BUICK_HTTPS_ADDR`**.

## Graceful shutdown

SIGINT and SIGTERM trigger a graceful shutdown with a 30 second deadline on **buickd**.
