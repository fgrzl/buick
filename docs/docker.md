# Docker and Compose

## Published image

Release workflow pushes multi-arch images to GitHub Container Registry, for example:

`ghcr.io/fgrzl/buick:latest`

Use your fork’s `ghcr.io/<owner>/<repo>:latest` if you do not use this upstream repository.

The image [defaults](../cmd/buickd/Dockerfile) to **`--config /etc/buick/buick.yml`**, so you do not need a **`command:`** override if you mount your config at that path.

## Minimal `docker-compose.yml`

Bind-mount a **config** and a **certs directory** so TLS material survives container recreation.

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

## Example `dev/buick.yml` (paths inside the container)

Use paths under **`/etc/buick/certs`**. For apps on the **host**, dial **`http://host.docker.internal:PORT`** — `127.0.0.1` from inside the container is not your laptop.

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

**`buickd`** listens on **80** / **443** here; upstreams use **8080**, **8081**, etc. on the host so they do not collide with the proxy ports.

If **`buickd`** runs in the **same** Compose project as your APIs, attach both to one user-defined network and use **`http://service1:8080`**-style targets (see repository [`buick.yml`](../buick.yml)). You usually do not need **`extra_hosts`** in that layout.

## Optional Compose snippets

| Situation | Add to the service |
|-----------|-------------------|
| Build from this repo instead of pulling | `build: { context: ., dockerfile: cmd/buickd/Dockerfile }` and drop or override **`image`**. |
| Linux Docker Engine, upstreams on the host via **`host.docker.internal`** | `extra_hosts: ["host.docker.internal:host-gateway"]` (Docker Desktop already defines **`host.docker.internal`**). |
| Upstream URL uses **`something.localhost`** and must resolve to the **host** from inside the container | `extra_hosts: ["something.localhost:host-gateway"]` (one entry per hostname you dial that way). |
| **`buickd` cannot write** TLS files into a bind-mounted cert dir (common on Linux with the **nonroot** image) | `user: "0:0"` for local dev only, or use a **named volume** for **`/etc/buick/certs`** instead of a host bind-mount. |

Listening on **80** / **443** inside the container is fine; if the **host** cannot bind low ports, use **`:8080` / `:8443`** in the YAML and map **`8080:8080`** (or similar) in **`ports`**.

## First run and trust

On the **host**, with the **`buick`** CLI installed and paths aligned with your Compose volume:

```bash
buick init --config ./dev/buick.yml
docker compose up -d
```

**`buick init`** creates any missing parent directories for **`cert_file`** and **`key_file`** before writing PEMs.

From a directory that contains **`buick.yml`**, **`buick init`** alone uses that file.

**Without `buick init`**, you can rely on **`buickd`**-generated self-signed material (trust the leaf or CA in the OS or Firefox) or use **mkcert** as described in [TLS and certificates](tls-and-certs.md).

## Integration stack in this repo

The repository **`compose.yml`** is for integration testing and sample backends, not the minimal “single `buick` service” layout above. See [Repository layout](repository-layout.md) and [Integration testing](testing.md).
