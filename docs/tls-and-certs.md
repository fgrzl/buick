# TLS and certificates

## `buick init` (recommended on the host)

Run **`buick init`** once to create a small local CA, issue a leaf for hostnames derived from your config, write PEMs to **`proxy.cert_file`** and **`proxy.key_file`**, and install the CA into the OS trust store where supported (Windows `certutil`, macOS `security` + login keychain, Debian-like `update-ca-certificates`). Parent directories for those PEM paths are created automatically if they do not exist.

- Pass **`--config /path/to/buick.yml`**, or run from a directory that contains **`buick.yml`** (default config file for `init` only).
- **`buick init -h`** documents **`--skip-trust`** (write PEMs only), **`--uninstall`**, and related flags.
- Use **`--skip-trust`** and import **`buick-root-ca.pem`** manually if trust installation fails.
- **Firefox NSS** is not modified; Chromium on Windows/macOS typically uses the OS store.

The CA PEM is written next to the leaf certificate (same directory as **`cert_file`**), named **`buick-root-ca.pem`** (see `internal/initca`).

## Behavior in `buickd` when HTTPS is enabled

With **`proxy.https`** set (including after listener defaults are applied), **`cert_file`** and **`key_file`** must each point to an **existing regular file**. **`buickd`** does not create or overwrite PEMs; use **`buick init`**, **mkcert**, or your own material.

Without trusting the CA or leaf, browsers will warn until you import trust or swap PEMs.

## mkcert

Alternative to `buick init`:

1. **`mkcert -install`** once.
2. Write leaf files matching **`cert_file`** / **`key_file`** in your YAML, for example:

   ```bash
   mkcert -cert-file ./dev/buick/certs/localhost.pem -key-file ./dev/buick/certs/localhost-key.pem \
     localhost 127.0.0.1 ::1 service1.localhost service2.localhost service3.localhost
   ```

3. Start **`buickd`**. Existing PEMs are not overwritten.

## Where files go

There is no separate “certs directory” setting. You choose directories via **`proxy.cert_file`** and **`proxy.key_file`**. The local CA from **`buick init`** is stored beside the leaf (**`dirname(cert_file)`**).

For Compose workflows that bind-mount a certs folder, see [Docker and Compose](docker.md).
