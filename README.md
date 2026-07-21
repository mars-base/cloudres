# cloudres

Cloud resource query and display tool with a TUI.

`cloudres` queries cloud provider CLIs (Aliyun, Huawei Cloud, etc.) and caches resources
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
| [Huawei Cloud (华为云)](https://www.huaweicloud.com/) | `hcloud` CLI + `~/.hcloud/config.json` | ✅ Implemented |

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
| `slb` | Server Load Balancer (CLB) | Address type, IP, network type, bandwidth, VPC, zone |
| `alb` | Application Load Balancer | Address type, edition, DNS, VPC |
| `nlb` | Network Load Balancer | Address type, DNS, VPC |
| `ess` | Auto Scaling (ESS) | Group type, active/max/min capacity |

### Resource Types (Huawei Cloud)

| Code | Resource | Detail Fields |
|------|----------|---------------|
| `ecs` | 弹性云服务器 (ECS) | vCPUs, RAM, OS, private/public IP, VPC, zone, flavor |
| `vpc` | 虚拟私有云 (VPC) | CIDR, description, created/updated time |
| `subnet` | 子网 (Subnet) | CIDR, VPC, gateway, zone, DHCP |
| `rds` | 云数据库 (RDS) | Engine, type, flavor, CPU, memory, volume, private/public IP, nodes |
| `dcs` | 分布式缓存服务 (DCS/Redis) | Engine, capacity, max/used memory, IP, port, VPC, zone |
| `evs` | 云硬盘 (EVS) | Size, type, bootable, encrypted, attachments |
| `eip` | 弹性公网IP (EIP) | Public IP, type, bandwidth, private IP, bound instance |

## Prerequisites

- [Go](https://go.dev/) 1.25+ (build from source)
- Provider-specific CLI tools (e.g. `aliyun` CLI for Alibaba Cloud, `hcloud` CLI for Huawei Cloud)
- A valid provider config (e.g. `~/.aliyun/config.json`, `~/.hcloud/config.json`)

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
  Provider  Profile         Regions
  ──────────────────────────────────────────
  aliyun    default         cn-hangzhou, cn-shanghai
  huawei    default          cn-east-3
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

# Huawei Cloud
cloudres huawei ecs --sync
cloudres huawei eip
cloudres huawei dcs --profile myproject --sync
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
| `:` | Command mode — type a resource type (e.g. `ecs`, `subnet`) |
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

Press `:` to enter command mode, then type a resource type name to load and display it:

| Input | Example | Action |
|-------|---------|--------|
| `<resource-type>` | `ecs` | Load ECS resources |
| `<resource-type>` | `subnet` | Load Subnet resources, hint shows `子网 (Subnet)` |
| `<resource-type>` | `dcs` | Load DCS (Redis) resources |

## License

[Apache License 2.0](LICENSE)
