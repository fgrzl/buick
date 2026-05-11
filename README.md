# buick

Small HTTP/HTTPS reverse proxy for local development. It routes by `Host`, forwards with `httputil.ReverseProxy`, supports WebSocket upgrades, and reads a YAML service table.

## Usage

Two binaries:

- **`buickd`** — HTTP/HTTPS reverse proxy (runs until stopped).
- **`buick`** — developer CLI: `buick init …`, `buick --check`, `buick --print-routes`.

Build and run the proxy:

```bash
go build -o buickd ./cmd/buickd
./buickd --config ./buick.yml
```

Build the CLI (same module):

```bash
go build -o buick ./cmd/buick
```

Validate configuration:

```bash
./buick --check --config ./buick.yml
```

Print the resolved routing table (normalized host to upstream URL):

```bash
./buick --print-routes --config ./buick.yml
```

## Configuration

Top-level fields:

- `proxy.http`: listen address for HTTP. Default in examples is **`:80`**. Omit to disable.
- `proxy.https`: listen address for HTTPS. Default in examples is **`:443`**. Omit to disable.
- `proxy.cert_file` / `proxy.key_file`: PEM paths used by the HTTPS listener. If **both** files exist they are reused; if **either** is missing, **buickd** generates a self-signed pair (see TLS below).

Each `services` entry maps a hostname to an upstream:

- `target`: absolute `http://` or `https://` URL for the upstream.
- `websocket`: when true, disables response buffering (`FlushInterval`) suitable for upgrades.
- `read_timeout` / `write_timeout`: optional `time.ParseDuration` strings. Defaults are 60s for normal HTTP and 168h for websocket services unless overridden.

Incoming requests are matched using `NormalizeHost(r.Host)` so `service1.localhost:80` matches `service1.localhost`. Unknown hosts receive `502 Bad Gateway`.

Forwarded headers set on the upstream request:

- `X-Forwarded-Host` (client `Host`, including port if present)
- `X-Forwarded-Proto` (`http` or `https`)
- `X-Forwarded-For` (appended client IP)

The original client `Host` header is preserved on the proxied request (not rewritten to the upstream hostname).

## TLS for local HTTPS

When `proxy.https` is set, **buickd** ensures PEM material exists at `cert_file` and `key_file`:

- If **both** files already exist, they are used as-is.
- If **either** file is missing, **buickd** generates a **self-signed** RSA certificate valid one year, with SANs for `localhost`, `127.0.0.1`, `::1`, and every hostname key under `services`.

Your OS or browser will warn until you trust the certificate (import the PEM into the system trust store, or use a tool like `mkcert` and replace the files).

## Docker Compose recipe

Buick does not ship a `compose.yml`; copy the pieces below into **your** project’s Compose file.

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

If **buickd** runs in the **same** Compose project as your APIs, put every service on one user-defined network and use **`http://service1:8080`** style targets instead of `host.docker.internal`.

### 3. First run and trust (low friction)

```bash
mkdir -p certs
docker compose up -d --build
```

**Trust once (pick one):**

1. **buickd-generated self-signed** — After the first successful start, import **`./certs/localhost.pem`** into your OS trust store (or Firefox’s Authorities). **buickd** reuses both PEMs when they already exist, so you do not repeat this unless you delete `./certs`.

2. **mkcert (fewest browser warnings)** — On the host: `mkcert -install` once, then create leaf files in `./certs` with the same names as `cert_file` / `key_file` in your buick config, for example:

```bash
mkcert -cert-file ./certs/localhost.pem -key-file ./certs/localhost-key.pem \
  localhost 127.0.0.1 ::1 service1.localhost service2.localhost service3.localhost
```

Then start Compose. **buickd** will **not** overwrite existing PEMs; trust comes from mkcert’s local CA.

## Integration tests (Docker)

The repo includes **`compose.yml`** with three [http-echo](https://github.com/hashicorp/http-echo) `1.0.0` backends (`service1`–`service3`) and a **buickd** service built from **`cmd/buickd/Dockerfile`**. **buickd** reads **`compose.buick.yml`** (HTTP **8080**, HTTPS **8443** inside the stack; published as **18080** / **18443** on the host). **buickd** runs as **`user: "0:0"`** in that file so it can write generated TLS material into the named `buick_certs` volume (distroless’s default nonroot user cannot create files there on a fresh volume).

```bash
docker compose up -d --build
BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
docker compose down
```

Override the default host URLs if needed: **`BUICK_HTTP_ADDR`**, **`BUICK_HTTPS_ADDR`**.

## Graceful shutdown

SIGINT and SIGTERM trigger a graceful shutdown with a 30 second deadline on **buickd**.
