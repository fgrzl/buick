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
      - "80:8080"
      - "443:8443"
    volumes:
      - ./dev/buick.yml:/etc/buick/buick.yml:ro
      - ./dev/buick/certs:/etc/buick/certs
```

## Example `dev/buick.yml` (paths inside the container)

Use paths under **`/etc/buick/certs`**. For apps on the **host**, dial **`http://host.docker.internal:PORT`** — `127.0.0.1` from inside the container is not your laptop.

```yaml
certs:
  path: "./dev/buick/certs"
proxy:
  http: ":8080"
  https: ":8443"
  certs_path: "/etc/buick/certs"

services:
  service1.localhost:
    target: "http://host.docker.internal:8080"
  service2.localhost:
    target: "http://host.docker.internal:8081"
```

**`buick init`** on the host writes **`localhost.pem`**, **`localhost-key.pem`**, and **`buick-root-ca.pem`** under **`certs.path`**. Mount that directory to **`/etc/buick/certs`** so **`buickd`** sees the same files at **`(proxy.certs_path)/localhost.pem`**.

Alternatively, mount PEMs you created with **mkcert** or another tool into **`proxy.certs_path`**.

**`buickd`** listens on **8080** / **8443** inside the container and publishes host **80** / **443**. The image runs as a non-root user, so avoid binding privileged ports inside the container unless you explicitly add capabilities or run it as root.

If **`buickd`** runs in the **same** Compose project as your APIs, attach both to one user-defined network and use **`http://service1:8080`**-style targets (see repository [`buick.yml`](../buick.yml)). You usually do not need **`extra_hosts`** in that layout.

## Optional Compose snippets

| Situation | Add to the service |
|-----------|-------------------|
| Build from this repo instead of pulling | `build: { context: ., dockerfile: cmd/buickd/Dockerfile }` and drop or override **`image`**. |
| Linux Docker Engine, upstreams on the host via **`host.docker.internal`** | `extra_hosts: ["host.docker.internal:host-gateway"]` (Docker Desktop already defines **`host.docker.internal`**). |
| Upstream URL uses **`something.localhost`** and must resolve to the **host** from inside the container | `extra_hosts: ["something.localhost:host-gateway"]` (one entry per hostname you dial that way). |
| **`buickd` exits** with TLS path errors | Ensure **`localhost.pem`** / **`localhost-key.pem`** exist under **`proxy.certs_path`** before start (run **`buick init`** on the host so **`certs.path`** matches your volume, use **mkcert**, or mount PEMs). **`buickd`** does not create PEMs. |

If the **host** cannot bind low ports, publish higher host ports instead, for example **`8080:8080`** and **`8443:8443`**.

## First run and trust

On the **host**, with the **`buick`** CLI installed: align **`certs.path`** with your bind-mounted cert directory and **`proxy.certs_path`** with paths **inside** the container (see **`dev/buick.yml`** above). Then run **`buick init`** and start Compose.

```bash
buick init --config ./dev/buick.yml
docker compose up -d
```

**`buick init`** creates any missing parent directories for **`certs.path`** before writing PEMs.

From a directory that contains **`buick.yml`**, **`buick init`** alone uses that file; otherwise pass **`--config`**.

**Without `buick init`**, supply PEMs yourself (for example **mkcert** as in [TLS and certificates](tls-and-certs.md)). **`buickd`** never writes certificate files.

## Integration stack in this repo

The repository **`compose.yml`** is for integration testing and sample backends, not the minimal “single `buick` service” layout above. See [Repository layout](repository-layout.md) and [Integration testing](testing.md).
