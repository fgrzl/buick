# Command-line reference

## `buickd`

| Invocation | Meaning |
|------------|---------|
| **`buickd --config PATH`** | Run the proxy until stopped; **`PATH`** is the Buick YAML file. |
| **`buickd --version`** | Print version, commit, and build date. |

There are no other flags today.

## `buick`

| Flag / subcommand | Meaning |
|--------------------|---------|
| **`--config PATH`** | Required for **`--check`** and **`--print-routes`** (not for **`buick init`**, which can default to **`buick.yml`** in the current directory). |
| **`--check`** | Load and validate the config; print **`config OK`** on success. |
| **`--print-routes`** | Print resolved host → upstream lines and exit. |
| **`--version`** | Print version information. |
| **`buick init`** | TLS/CA setup; see **`buick init -h`** for **`--config`**, **`--skip-trust`**, **`--uninstall`**, etc. |

With no flags except **`--config`**, **`buick`** exits with a usage hint (use **`--check`**, **`--print-routes`**, or **`buick init`**).
