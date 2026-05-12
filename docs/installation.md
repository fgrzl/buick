# Installation

Requires [Go](https://go.dev/dl/) **1.22+** (matches the `go` directive in this module’s `go.mod`).

## Install the `buick` CLI

**From the public module** (binary name `buick`):

```bash
go install github.com/fgrzl/buick/cmd/buick@latest
```

Put Go’s install directory on your **`PATH`** (often `$HOME/go/bin` on Linux/macOS, `%USERPROFILE%\go\bin` on Windows), then:

```bash
buick --version
```

**From a clone** (whatever revision you have checked out):

```bash
go install ./cmd/buick
```

**Build CLI only** (writes `buick` in the current directory):

```bash
go build -o buick ./cmd/buick
```

### Private modules

If `go install` cannot fetch the module, set [`GOPRIVATE`](https://go.dev/ref/mod#private-modules) and Git/SSH, or install from a local clone as above.

## Standalone daemon

Run **`buickd`** on your machine without Docker (after **`buick init`** if you use HTTPS paths from the config).

From a clone:

```bash
go build -o buickd ./cmd/buickd
./buickd --config ./path/to/buick.yml
```

- **[`buick.host.example.yml`](../buick.host.example.yml)** — upstreams on **`127.0.0.1`** (typical laptop dev).
- **[`buick.yml`](../buick.yml)** — Compose-style upstream hostnames when **`buickd`** shares a Docker network with backends.

Sanity-check YAML before starting:

```bash
buick --check --config ./path/to/buick.yml
```

Prebuilt **`buickd`** images: [Docker and Compose](docker.md).
