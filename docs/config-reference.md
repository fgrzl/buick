# Configuration reference

Buick reads one YAML file. The top level has **`proxy`** (listeners and TLS paths) and **`services`** (hostname → upstream).

## `proxy`

| Field | Meaning |
|-------|---------|
| **`http`** | HTTP listen address (e.g. **`:80`**). |
| **`https`** | HTTPS listen address (e.g. **`:443`**). |
| **`cert_file` / `key_file`** | Paths to the TLS leaf certificate and private key. If **both** exist they are reused; if **either** is missing, **`buickd`** can generate a dev self-signed pair (see [TLS and certificates](tls-and-certs.md)). |

### Listener defaults

If **`http`** and **`https`** are both omitted:

- **`buickd`** defaults **`http`** to **`:80`**.
- If **`cert_file`** and **`key_file`** are both set, **`https`** defaults to **`:443`**, so a minimal file can list only PEM paths plus **`services`**.

If **`http`** is set but **`https`** is omitted and both PEM paths are set, **`https`** defaults to **`:443`**.

## `services` (per hostname)

Each key is a hostname (or hostname with port; see matching below). The value describes how to reach the upstream.

| Field | Meaning |
|-------|---------|
| **`target`** | Single absolute `http://` or `https://` URL **`buickd`** dials. Use **Compose service names** (e.g. `http://api:8080`) when **`buickd`** shares a Docker network with the backend; **`http://host.docker.internal:PORT`** when the app runs on the **host** and **`buickd`** is in a container; **`http://127.0.0.1:PORT`** when **`buickd`** runs on the **same machine** as the backend. |
| **`targets`** | List of upstream URLs for **round-robin** on the same hostname. Do **not** set **`target`** in the same entry. **No health checks** — peers rotate regardless of availability. |
| **`websocket`** | If true, suitable `FlushInterval` for WebSocket upgrades. |
| **`read_timeout` / `write_timeout`** | Optional `time.ParseDuration`. Defaults: **60s** HTTP, **168h** when `websocket: true`. |

### Matching

Incoming `Host` is normalized (for example `service1.localhost:80` matches `service1.localhost`). Unknown hosts receive **502 Bad Gateway**.

### Forwarded headers

On the upstream request, Buick sets **`X-Forwarded-Host`**, **`X-Forwarded-Proto`**, and **`X-Forwarded-For`** (client IP appended). The client **`Host`** header is **not** rewritten to the upstream hostname.

## Example shapes

- **Minimal with TLS paths** (listeners default to `:80` / `:443`): see repository [`buick.yml`](../buick.yml).
- **Host-only upstreams**: see [`buick.host.example.yml`](../buick.host.example.yml) in the repo root.
- **Integration stack** (internal `:8080` / `:8443`): see [`tests/integration/buick.yml`](../tests/integration/buick.yml).
