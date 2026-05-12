# buick

Small **HTTP/HTTPS reverse proxy** for **local development** — route by `Host`, YAML config, optional WebSockets, TLS helpers.

**[Full documentation →](docs/README.md)**

## Install the CLI

Requires [Go](https://go.dev/dl/) **1.22+**.

```bash
go install github.com/fgrzl/buick/cmd/buick@latest
```

Put Go’s **`bin`** on your **`PATH`** (for example `$HOME/go/bin` on Linux/macOS, `%USERPROFILE%\go\bin` on Windows).

## TLS (`buick init`)

Add a Buick YAML with **`proxy.cert_file`** and **`proxy.key_file`** (see [Configuration](docs/config-reference.md)). **`buick init`** creates missing parent directories for those paths, writes the leaf and local CA, and installs trust where the OS supports it.

```bash
buick init --config ./dev/buick.yml
```

If **`buick.yml`** sits in the current directory, **`buick init`** alone picks it up. Use **`buick init -h`** for **`--skip-trust`**, **`--uninstall`**, and other flags.

## Docker Compose

The image runs **`buickd`** with the default **`--config /etc/buick/buick.yml`**. Mount your config and a certs directory (PEMs from **`buick init`** on the host):

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

```bash
docker compose up -d
```

Image tags, **`extra_hosts`**, and a fuller **`dev/buick.yml`** example: [Docker and Compose](docs/docker.md). Running **`buickd`** on the host without containers: [Standalone daemon](docs/installation.md#standalone-daemon).
