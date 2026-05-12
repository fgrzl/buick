# buick

Small **HTTP/HTTPS reverse proxy** for **local development**: route by `Host`, forward with `httputil.ReverseProxy`, optional **WebSockets**, YAML service table, **TLS** helpers, and **loopback-only** management endpoints.

| Binary | Role |
|--------|------|
| **`buickd`** | Runs the proxy until stopped (`--config` required). |
| **`buick`** | Validate config, print routes, TLS/CA setup (`buick init`), `--version`. |

---

## Install the `buick` CLI

Requires [Go](https://go.dev/dl/) **1.22+** (matches this module).

**From the module path** (binary name `buick`):

```bash
go install github.com/fgrzl/buick/cmd/buick@latest
```

Ensure Go’s install dir is on your **`PATH`** (often `$HOME/go/bin` on Linux/macOS, `%USERPROFILE%\go\bin` on Windows), then:

```bash
buick --version
```

**From a clone** (whatever revision you have checked out):

```bash
go install ./cmd/buick
```

**Build CLI only** (writes `buick` in the current directory):

```bash
go build -o buick ./cmd/buick
```

**Private modules:** If `go install` cannot fetch the module, set [`GOPRIVATE`](https://go.dev/ref/mod#private-modules) and Git/SSH, or install from a local clone as above.

---

## Quick start

1. **Build the daemon** (from a clone):

   ```bash
   go build -o buickd ./cmd/buickd
   ```

2. **Pick a config** — on the **host**, use something like **`buick.host.example.yml`** with `target: "http://127.0.0.1:…"`. The sample **`buick.yml`** expects Docker DNS names (`service1`, …); running **`buickd --config ./buick.yml`** on the laptop without that network will not reach those upstreams.

3. **Run:**

   ```bash
   ./buickd --config ./buick.host.example.yml
   ```

4. **Optional TLS on the host:** `buick init` with `--config …`, or run **`buick init`** from a directory that contains **`buick.yml`** (see [TLS for local HTTPS](#tls-for-local-https)). Use `buick init -h` for `--skip-trust`, `--uninstall`, etc.

5. **Sanity check:**

   ```bash
   buick --check --config ./buick.host.example.yml
   buick --print-routes --config ./tests/integration/buick.yml
   ```

---

## Repo config files

| File | Purpose |
|------|---------|
| **`compose.yml`** | Integration stack: nginx backends `service1`–`service3`, **buickd** on `buick-integration`, host ports **18080** (HTTP) / **18443** (HTTPS). |
| **`tests/integration/buick.yml`** | **buickd** config for the integration stack: **`:8080` / `:8443`**, certs under **`/etc/buick/certs/`**, upstreams like **`http://service1:8080`**. |
| **`buick.yml`** | Minimal sample: **`cert_file`** / **`key_file`** under **`./dev/buick/certs/`**, **`services`** with Compose-style upstreams; **`http`**/**`https`** default to **`:80`** / **`:443`** when omitted. |
| **`buick.host.example.yml`** | **buickd on the host**, upstreams on **`127.0.0.1`**. Copy or merge into your own file. |

### Sample stack (Docker)

From the repo root:

```bash
docker compose up -d --build
```

Then use **http://127.0.0.1:18080/** with `Host: service1.localhost`, or add **`127.0.0.1 service1.localhost`** to your hosts file and open the hostname in the browser.

---

## Configuration

### `proxy`

| Field | Meaning |
|-------|---------|
| **`http`** | HTTP listen address (e.g. **`:80`**). |
| **`https`** | HTTPS listen address (e.g. **`:443`**). |
| **`cert_file` / `key_file`** | PEM paths for HTTPS. If **both** exist they are reused; if **either** is missing, **buickd** can generate a dev self-signed pair (see [TLS](#tls-for-local-https)). |

If **`http`** and **`https`** are both omitted, **`buickd`** defaults **`http`** to **`:80`**. If **`cert_file`** and **`key_file`** are both set, **`https`** also defaults to **`:443`**, so a minimal file can list only PEM paths plus **`services`**. If **`http`** is set but **`https`** is omitted and both PEM paths are set, **`https`** defaults to **`:443`**.

### `services` (per hostname)

| Field | Meaning |
|-------|---------|
| **`target`** | Single absolute `http://` or `https://` URL **buickd** dials. Use **Compose service names** (e.g. `http://api:8080`) when **buickd** shares a network with the backend; **`http://host.docker.internal:PORT`** when the app is on the **host** and **buickd** is in a container; **`http://127.0.0.1:PORT`** when **buickd** runs on the **same machine** as the backend. |
| **`targets`** | List of upstream URLs for **round-robin** on the same hostname. Do **not** set **`target`** in the same entry. **No health checks** — peers rotate regardless of availability. |
| **`websocket`** | If true, suitable `FlushInterval` for WebSocket upgrades. |
| **`read_timeout` / `write_timeout`** | Optional `time.ParseDuration`. Defaults: **60s** HTTP, **168h** when `websocket: true`. |

**Matching:** Incoming `Host` is normalized (`service1.localhost:80` matches `service1.localhost`). Unknown hosts → **502 Bad Gateway**.

**Forwarded headers** on the upstream request: `X-Forwarded-Host`, `X-Forwarded-Proto`, `X-Forwarded-For` (client IP appended). The client **`Host`** header is **not** rewritten to the upstream hostname.

---

## Operations

### Management (`/_buick/*`, loopback only)

On **both** HTTP and HTTPS listeners, these **GET** endpoints work only when the **TCP client is loopback** (`127.0.0.1`, `::1`, …) — judged from **`RemoteAddr`**:

| Path | Response |
|------|----------|
| **`/_buick/health`** | JSON: status, version, uptime, route **count** (no full route table). |
| **`/_buick/routes`** | JSON array of routes, **sorted by hostname** (stable for scripts and diffs). |
| **`/_buick/metrics`** | Prometheus-style text (404 if metrics disabled). |

Unknown paths under **`/_buick/...`** → **404** on loopback (e.g. typos).

If **buickd** sits **behind another reverse proxy**, these URLs usually **do not** hit the mgmt handlers unless you talk to **buickd** directly on loopback or the proxy preserves loopback source addresses.

### Reload (`SIGHUP`)

Send **`SIGHUP`** to reload the route table from the same **`--config`** file (common on Unix; on Windows this may not be available). If load or validation fails, **old routes stay active**.

**Not** reloaded until process restart: **listener addresses** and **server read/write timeouts**. TLS leaf files may still be refreshed when HTTPS is enabled (same dev-cert behavior as startup). Successful reload does not spam “using existing TLS material” logs when certs are unchanged.

### Shutdown

**SIGINT** / **SIGTERM** → graceful shutdown (**30s** deadline per server).

---

## TLS for local HTTPS

**Recommended on the host:** run **`buick init`** once (with **`--config`** or a default **`buick.yml`** in the current directory). It creates a small local CA, issues a leaf for hostnames in your YAML, writes `cert_file` / `key_file`, and installs the CA via **OS tools** (Windows `certutil`, macOS `security` + keychain, Debian-like `update-ca-certificates`). Use **`--skip-trust`** and import **`buick-root-ca.pem`** manually if install fails. **Firefox NSS** is not modified; Chromium on Windows/macOS usually uses the OS store.

With **`proxy.https`** set, **buickd** ensures PEMs exist at **`cert_file`** / **`key_file`**:

- **Both** files present → used as-is (**`buick init`**, **mkcert**, etc.).
- **Either** missing → **buickd** generates a self-signed RSA leaf (1 year), SANs for `localhost`, `127.0.0.1`, `::1`, and every **`services`** hostname.

Without trusting the CA or leaf, browsers will warn until you import trust or swap PEMs (e.g. **mkcert** below).

---

## Docker Compose recipe

This repo ships **`compose.yml`** and **`tests/integration/buick.yml`** for the [integration stack](#integration-tests-docker). For day-to-day use, a published image is enough: the image [defaults](cmd/buickd/Dockerfile) to **`--config /etc/buick/buick.yml`**, so you do not need a **`command:`** line if you mount the config there.

### Minimal `docker-compose.yml`

Replace the image with your registry tag if you use a fork or pin by digest. Bind-mount a **config** and a **certs directory** so TLS material survives container restarts.

```yaml
services:
  buick:
    image: ghcr.io/fgrzl/buick:latest
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./dev/buick.yml:/etc/buick/buick.yml:ro
      - ./dev/buick/certs:/etc/buick/certs
```

### Example `dev/buick.yml` (paths inside the container)

Use **paths under `/etc/buick/certs`**. For apps on the **host**, dial **`http://host.docker.internal:PORT`** (`127.0.0.1` from inside the container is not your laptop).

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

**buickd** listens on **80** / **443** here; upstreams use **8080**, **8081**, etc. on the host so nothing collides with the proxy ports.

If **buickd** runs in the **same** Compose project as your APIs, attach both to one user-defined network and use **`http://service1:8080`**-style targets (see **`buick.yml`**). Then you usually do not need **`extra_hosts`**.

### Optional Compose snippets

| Situation | Add to the service |
|-----------|-------------------|
| Build from this repo instead of pulling | `build: { context: ., dockerfile: cmd/buickd/Dockerfile }` and drop or override **`image`**. |
| Linux Docker Engine, upstreams on the host via **`host.docker.internal`** | `extra_hosts: ["host.docker.internal:host-gateway"]` (Docker Desktop already defines **`host.docker.internal`**). |
| Upstream URL uses **`something.localhost`** and must resolve to the **host** from inside the container | `extra_hosts: ["something.localhost:host-gateway"]` (one entry per hostname you dial that way). |
| **`buickd` cannot write** TLS files into a bind-mounted cert dir (common on Linux with the **nonroot** image) | `user: "0:0"` for local dev only, or use a **named volume** for **`/etc/buick/certs`** instead of a host bind-mount. |

Listening on **80** / **443** inside the container is fine; if your **host** cannot bind low ports, use **`:8080` / `:8443`** in the YAML and map **`8080:8080`** in **`ports`**.

### First run and trust

```bash
mkdir -p dev/buick/certs
```

On the **host**, with **`buick`** [installed](#install-the-buick-cli), align paths with your Compose volume, then:

```bash
buick init --config ./dev/buick.yml
docker compose up -d
```

From a directory that contains **`buick.yml`**, **`buick init`** alone uses that file.

**Without `buick init`**, pick one:

1. **buickd-generated self-signed** — After first start, trust **`./dev/buick/certs/localhost.pem`** (or your configured filenames) in the OS or Firefox. Existing PEMs are reused until deleted.

2. **mkcert** — `mkcert -install` once, then write leaf files matching `cert_file` / `key_file`:

   ```bash
   mkcert -cert-file ./dev/buick/certs/localhost.pem -key-file ./dev/buick/certs/localhost-key.pem \
     localhost 127.0.0.1 ::1 service1.localhost service2.localhost service3.localhost
   ```

   Then start Compose. **buickd** does not overwrite existing PEMs.

---

## Integration tests (Docker)

**`compose.yml`** runs three **nginx** backends (`service1`–`service3`; configs under **`tests/integration/nginx/`**) and **buickd** from **`cmd/buickd/Dockerfile`**. **buickd** loads **`tests/integration/buick.yml`** (internal **8080** / **8443**; published **18080** / **18443**). **buickd** runs as **`user: "0:0"`** there so it can write TLS into the **`buick_certs`** volume on first boot.

```bash
docker compose up -d --build
BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
docker compose down
```

With **`BUICK_INTEGRATION=1`**, tests wait up to **90s** for **http://127.0.0.1:18080**. Override probes with **`BUICK_HTTP_ADDR`** and **`BUICK_HTTPS_ADDR`** if needed.
