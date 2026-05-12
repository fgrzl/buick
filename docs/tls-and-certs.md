# TLS and certificates

## `buick init` (recommended on the host)

Run **`buick init`** once to create a small local CA, issue a leaf for hostnames derived from your config, write PEMs under **`certs.path`** (see [Configuration](config-reference.md)), and install the CA into the OS trust store where supported (Windows `certutil`, macOS `security` + login keychain, Debian-like `update-ca-certificates`). Parent directories are created automatically if they do not exist.

- **`buick init`** writes **`localhost.pem`**, **`localhost-key.pem`**, and **`buick-root-ca.pem`** under **`certs.path`** (default **`./buick/certs`** when TLS is enabled and **`certs.path`** is omitted).
- **`buickd`** reads **`localhost.pem`** and **`localhost-key.pem`** from **`proxy.certs_path`** (default **`./dev/buick/certs`** when omitted). Paths are not listed as PEM filenames in YAML.
- Pass **`--config /path/to/buick.yml`**, or run from a directory that contains **`buick.yml`** (default for **`buick init`** only).
- **`buick init -h`** documents **`--skip-trust`** (write PEMs only), **`--uninstall`**, and related flags.
- Use **`--skip-trust`** and import **`buick-root-ca.pem`** manually if trust installation fails.
- **Firefox NSS** is not modified; Chromium on Windows/macOS typically uses the OS store.

The CA PEM is written next to the leaf (under **`certs.path`**), named **`buick-root-ca.pem`** (see `internal/initca`).

## Behavior in `buickd` when HTTPS is enabled

With **`proxy.https`** set (including after listener defaults are applied), **`localhost.pem`** and **`localhost-key.pem`** must exist under **`proxy.certs_path`**. **`buickd`** does not create or overwrite PEMs; use **`buick init`**, **mkcert**, or your own material placed in that directory.

Without trusting the CA or leaf, browsers will warn until you import trust or swap PEMs.

## mkcert

Alternative to `buick init`:

1. **`mkcert -install`** once.
2. Write **`localhost.pem`** and **`localhost-key.pem`** into **`proxy.certs_path`** (and align **`certs.path`** if you still use **`buick init`** for the CA), for example:

   ```bash
   mkcert -cert-file ./dev/buick/certs/localhost.pem -key-file ./dev/buick/certs/localhost-key.pem \
     localhost 127.0.0.1 ::1 service1.localhost service2.localhost service3.localhost
   ```

3. Start **`buickd`**. Existing PEMs are not overwritten.

## Where files go

**`buickd`** loads TLS only from **`(proxy.certs_path)/localhost.pem`** and **`…/localhost-key.pem`**. **`buick init`** writes under **`(certs.path)/`**. For Compose, see [Docker and Compose](docker.md).
