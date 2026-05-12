# Runtime behavior

## Management API (`/_buick/*`, loopback only)

On **both** HTTP and HTTPS listeners, these **GET** endpoints respond only when the **TCP client is loopback** (`127.0.0.1`, `::1`, …), determined from **`RemoteAddr`**:

| Path | Response |
|------|------------|
| **`/_buick/health`** | JSON: status, version, uptime, route **count** (not the full route table). |
| **`/_buick/routes`** | JSON array of routes, **sorted by hostname** (stable for scripts and diffs). |
| **`/_buick/metrics`** | Prometheus-style text (404 if metrics are disabled). |

Unknown paths under **`/_buick/...`** return **404** on loopback (for example typos).

If **`buickd`** sits **behind another reverse proxy**, these URLs usually **do not** reach the management handlers unless you talk to **`buickd`** directly on loopback or the front proxy preserves loopback source addresses.

## Reload (`SIGHUP`)

Send **`SIGHUP`** to reload the route table from the same **`--config`** file (common on Unix; on Windows this may not be available). If load or validation fails, **old routes stay active**.

**Not** reloaded until process restart: **listener addresses** and **server read/write timeouts**. On **`SIGHUP`**, **`buickd`** re-checks that **`cert_file`** and **`key_file`** still exist when HTTPS is enabled; it does not write or rotate PEMs.

## Shutdown

**SIGINT** / **SIGTERM** trigger graceful shutdown (**30s** deadline per server).
