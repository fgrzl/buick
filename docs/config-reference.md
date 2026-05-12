# Configuration reference

Buick reads one YAML file with **`certs`** (optional, where **`buick init`** writes), **`proxy`** (listeners and where **`buickd`** reads TLS files), and **`services`**.

## `certs` (optional)

| Field | Meaning |
|-------|---------|
| **`path`** | Directory on the machine where you run **`buick init`**. Files are always **`localhost.pem`**, **`localhost-key.pem`**, and **`buick-root-ca.pem`** (beside the leaf). Default **`./buick/certs`** when TLS is enabled and this field is omitted. **`buickd` does not read this block.** |

## `proxy`

| Field | Meaning |
|-------|---------|
| **`http`** | HTTP listen address. Default **`:80`** when omitted and TLS applies, or when both **`http`** and **`https`** are omitted with TLS (see listener defaults). |
| **`https`** | HTTPS listen address. Default **`:443`** when omitted if TLS applies. TLS is enabled when **`https`** is set in YAML, or when **`proxy.certs_path`** or **`certs.path`** is set (even to a custom directory). |
| **`certs_path`** | Directory **`buickd`** loads **`localhost.pem`** and **`localhost-key.pem`** from. Default **`./dev/buick/certs`** when TLS applies and omitted. |

TLS file paths are **never** listed as PEM paths in YAML: Buick derives **`(proxy.certs_path)/localhost.pem`** and **`(proxy.certs_path)/localhost-key.pem`** internally after load.

### Listener defaults

If **`http`** and **`https`** are both omitted:

- **`buickd`** defaults **`http`** to **`:80`**.
- If TLS applies (**`https`** set, or **`proxy.certs_path`**, or **`certs.path`** present in YAML), **`https`** defaults to **`:443`**.

If **`http`** is set but **`https`** is omitted and TLS applies, **`https`** defaults to **`:443`**.

## `services` (per hostname)

Each key is a hostname (or hostname with port; see matching below). The value describes how to reach the upstream.

| Field | Meaning |
|-------|---------|
| **`target`** | Single absolute `http://` or `https://` URL **`buickd`** dials. Use **Compose service names** (e.g. `http://api:8080`) when **`buickd`** shares a Docker network with the backend; **`http://host.docker.internal:PORT`** when the app runs on the **host** and **`buickd`** is in a container; **`http://127.0.0.1:PORT`** when **`buickd`** runs on the **same machine** as the backend. |
| **`targets`** | List of upstream URLs for **round-robin** on the same hostname. Do **not** set **`target`** in the same entry. **No health checks** — peers rotate regardless of availability. |
| **`read_timeout` / `write_timeout`** | Optional `time.ParseDuration`. When omitted, per-route defaults are **60s** (shown in **`/_buick/routes`**). The **HTTP server** read/write deadlines are the maximum across routes, with a **168h** minimum so WebSocket upgrades and long-lived streams work without extra YAML. |

### WebSockets

**`Upgrade: websocket`** requests are forwarded automatically. Tune **`read_timeout`** / **`write_timeout`** only when you need non-default per-route values for metrics and documentation; the **168h** server floor still applies.

### Matching

Incoming `Host` is normalized (for example `service1.localhost:80` matches `service1.localhost`). Unknown hosts receive **502 Bad Gateway**.

### Forwarded headers

On the upstream request, Buick sets **`X-Forwarded-Host`**, **`X-Forwarded-Proto`**, and **`X-Forwarded-For`** (client IP appended). The client **`Host`** header is **not** rewritten to the upstream hostname.

## Example shapes

- **Same host for init + daemon**: set **`certs.path`** and **`proxy.certs_path`** to the same directory (see repository [`buick.yml`](../buick.yml)).
- **Docker**: host **`certs.path`** vs container **`proxy.certs_path`** (see [Docker and Compose](docker.md)).
- **Host-only upstreams**: [`buick.host.example.yml`](../buick.host.example.yml).
- **Integration stack**: [`tests/integration/buick.yml`](../tests/integration/buick.yml).
