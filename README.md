# cloudres

Cloud resource query and display tool with a TUI.

`cloudres` queries cloud provider CLIs (Aliyun, AWS, etc.) and caches resources
in a local SQLite database. It provides both a CLI interface for scripting and
an interactive terminal UI for browsing resources across regions.

## Features

- **Interactive TUI** — Browse providers, regions, and resources with
  vim-style keyboard navigation (`j`/`k`, `/` filter, `:` command)
- **CLI mode** — Tabulated output for use in scripts and pipelines
- **Local SQLite cache** — Resources are synced once and cached, with
  on-demand refresh (`--sync` flag)
- **Extensible provider model** — New cloud providers plug into the
  `ProviderDetector` + `ResourceFetcher` interface
- **Cross-platform** — Linux, macOS, Windows (amd64 + arm64)

## Supported Providers

| Provider | Detection | Status |
|----------|-----------|--------|
| [Aliyun (阿里云)](https://www.aliyun.com/) | `aliyun` CLI + `~/.aliyun/config.json` | ✅ Implemented |

### Resource Types (Aliyun)

| Code | Resource | Detail Fields |
|------|----------|---------------|
| `ecs` | Elastic Compute Service | CPU, memory, OS, private/public IP, VPC, zone |
| `vpc` | Virtual Private Cloud | CIDR, IPv6 CIDR |
| `vsw` | VSwitch (Subnet) | CIDR, zone, VPC |
| `rds` | Relational Database Service | Engine, class, storage, disk/backup size, endpoint |
| `tair` | Tair (Redis-compatible) | Type, edition, version, memory usage, quota |
| `pdb` | PolarDB | Engine, node specs (per-node role/CPU/memory), endpoints, storage |
| `oss` | Object Storage Service | Storage class |

## Prerequisites

- [Go](https://go.dev/) 1.25+ (build from source)
- Provider-specific CLI tools (e.g. `aliyun` CLI for Alibaba Cloud)
- A valid provider config (e.g. `~/.aliyun/config.json` with profiles)

## Installation

### Build from source

```bash
git clone https://github.com/mars-base/cloudres.git
cd cloudres
make build
./build/cloudres
```

### Install from GitHub releases (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | bash -s v1.0.0
```

Custom install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | INSTALL_DIR=~/bin bash
```

### Install from GitHub releases (Windows PowerShell)

```powershell
Invoke-WebRequest -Uri https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.ps1 -OutFile install.ps1
.\install.ps1

# Specific version
.\install.ps1 -Tag v1.0.0

# Custom directory
.\install.ps1 -InstallDir C:\tools
```

### Install via `go install`

```bash
go install github.com/mars-base/cloudres/cmd/cloudres@latest
```

### Pre-built binaries

```bash
make release
# Output in build/:
#   cloudres-linux-amd64   cloudres-linux-arm64
#   cloudres-darwin-amd64  cloudres-darwin-arm64
#   cloudres-windows-amd64.exe
```

## CLI Usage

### Enter TUI (default)

```bash
cloudres
```

### List detected providers

```bash
cloudres list
```

Output:

```
  Provider  Profile  Regions
  ─────────────────────────────────────
  aliyun    default  cn-hangzhou, cn-shanghai
```

### Query resources from CLI

```bash
# List ECS instances (uses current profile from config)
cloudres aliyun ecs

# Use a specific profile
cloudres aliyun ecs --profile production

# List resources in a specific region
cloudres aliyun ecs --region cn-hangzhou

# Force re-sync before listing
cloudres aliyun pdb --sync
```

### Show version

```bash
cloudres version
```

## TUI Key Bindings

### Provider Selection (`StateProviderSelect`)

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate |
| `Enter` | Select provider & profile |
| `:` | Enter command mode |
| `q` | Quit |

### Main View (`StateMain`)

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate resources |
| `1` – `9` | Select region by index |
| `:` | Command mode — type a resource type (e.g. `ecs`) or region |
| `/` | Filter resources (case-insensitive substring search across all columns) |
| `d` | View resource detail |
| `Esc` | Go back (clear filter → clear resource type → provider select) |
| `q` | Quit |

### Detail View (`StateDetail`)

| Key | Action |
|-----|--------|
| `Esc` / `d` | Back to main view |
| `q` | Quit |

### Command Mode

Press `:` to enter, then type:

| Input | Example | Action |
|-------|---------|--------|
| `<provider>` | `aliyun` | Select provider |
| `<provider>(<profile>)` | `aliyun(default)` | Select provider + profile |
| `<region>` | `cn-hangzhou` | Select region |
| `<resource-type>` | `ecs` | Select resource type and fetch |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint (requires golangci-lint)
make lint

# Cross-compile release binaries
make release
```

## License

[Apache License 2.0](LICENSE)
