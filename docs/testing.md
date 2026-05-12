# Integration testing

The **`compose.yml`** stack runs three **nginx** backends (`service1`–`service3`; configs under **`tests/integration/nginx/`**) and **`buickd`** built from **`cmd/buickd/Dockerfile`**. **`buickd`** loads **`tests/integration/buick.yml`** (internal **8080** / **8443**; published **18080** / **18443**). **`buickd`** runs as **`user: "0:0"`** so it can write TLS into the **`buick_certs`** named volume on first boot.

## Commands

```bash
docker compose up -d --build
BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
docker compose down
```

With **`BUICK_INTEGRATION=1`**, tests wait up to **90s** for **http://127.0.0.1:18080**.

Override probe addresses with **`BUICK_HTTP_ADDR`** and **`BUICK_HTTPS_ADDR`** if needed.

Default **`go test ./...`** compiles the integration package but skips docker-backed tests unless **`BUICK_INTEGRATION`** is set.
