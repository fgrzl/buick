# Repository layout

## Config and Compose files

| File | Purpose |
|------|---------|
| **`compose.yml`** | Integration stack: nginx backends `service1`–`service3`, **`buickd`** on Docker network `buick-integration`, host ports **18080** (HTTP) / **18443** (HTTPS). |
| **`tests/integration/buick.yml`** | **`buickd`** config for that stack: internal **`:8080` / `:8443`**, certs under **`/etc/buick/certs/`**, upstreams like **`http://service1:8080`**. |
| **`tests/integration/certs/`** | Fixture **`localhost.pem`** / **`localhost-key.pem`** for HTTPS in the stack (regenerate: **`go run ./tests/integration/gencerts`**). |
| **`buick.yml`** | Minimal sample: **`cert_file`** / **`key_file`** under **`./dev/buick/certs/`**, **`services`** with Compose-style upstreams; **`http`** / **`https`** default to **`:80`** / **`:443`** when omitted. |
| **`buick.host.example.yml`** | Example for **`buickd` on the host** with upstreams on **`127.0.0.1`**. Copy or merge into your own file. |

## Trying the sample stack

From the repository root:

```bash
docker compose up -d --build
```

Then use **http://127.0.0.1:18080/** with `Host: service1.localhost`, or add **`127.0.0.1 service1.localhost`** to your hosts file and open the hostname in the browser.

Automated integration tests are documented in [Integration testing](testing.md).
