# Overview

Buick helps you route HTTP(S) traffic by hostname to local or containerized backends during development, without running a full production ingress stack.

## Binaries

| Binary | Role |
|--------|------|
| **`buickd`** | Runs the proxy until stopped. Requires **`--config /path/to/buick.yml`**. |
| **`buick`** | Developer CLI: validate config (`--check`), print routes (`--print-routes`), TLS/CA setup (`buick init`), **`--version`**. |

The daemon loads a single YAML file. The CLI reads the same format for checks, route dumps, and `buick init` (which writes PEM material to the paths declared in `proxy.cert_file` and `proxy.key_file`).

See [Configuration](config-reference.md) for the full schema and [Installation](installation.md) for how to obtain the binaries.
