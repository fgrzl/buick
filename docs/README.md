# Buick documentation

Buick is a small **HTTP/HTTPS reverse proxy** for **local development**: route by `Host`, forward with `httputil.ReverseProxy`, optional **WebSockets**, a YAML route table, TLS helpers, and **loopback-only** management endpoints.

| Guide | Contents |
|--------|-----------|
| [Overview](overview.md) | Binaries (`buick` vs `buickd`), what each tool does |
| [CLI reference](cli.md) | Flags for `buick` and `buickd` |
| [Installation](installation.md) | Install the **`buick`** CLI, private modules, **standalone `buickd`** (build & run) |
| [Configuration](config-reference.md) | YAML: `proxy`, `services`, defaults, matching, forwarded headers |
| [TLS and certificates](tls-and-certs.md) | `buick init`, trust stores, `buickd` dev certs, mkcert |
| [Docker and Compose](docker.md) | Images, minimal Compose, host vs container upstreams, optional snippets |
| [Runtime behavior](runtime.md) | `/_buick/*` management API, reload (`SIGHUP`), shutdown |
| [Repository layout](repository-layout.md) | Sample configs in this repo, integration Docker stack |
| [Integration testing](testing.md) | `BUICK_INTEGRATION`, `docker compose`, probe env vars |

Start here: the repo **[README](../README.md)** covers **`go install`**, **`buick init`**, and minimal Docker Compose. Then [Configuration](config-reference.md) and [Docker and Compose](docker.md) for details.
