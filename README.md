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
- **Alternate CLI config file** — Global `-c`/`--config` flag points at a
  non-default vendor CLI config; the vendor is auto-detected and the run is
  restricted to it (Aliyun only)
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
| `ecs` | Elastic Compute Service | CPU, memory, OS, private/public IP, VPC, zone, auto-renewal (PrePaid) |
| `vpc` | Virtual Private Cloud | CIDR, IPv6 CIDR |
| `vsw` | VSwitch (Subnet) | CIDR, zone, VPC |
| `rds` | Relational Database Service | Engine, class, storage, disk/backup size, endpoint, whitelist groups (ACL) with IP lists |
| `tair` | Tair (Redis-compatible) | Type, edition, version, memory usage, quota |
| `pdb` | PolarDB | Engine, node specs (per-node role/CPU/memory), endpoints, storage, whitelist groups (ACL) with IP lists |
| `oss` | Object Storage Service | Storage class |
| `slb` | Server Load Balancer (CLB) | Address type, IP, network type, bandwidth, VPC, zone |
| `alb` | Application Load Balancer | Address type, edition, DNS, VPC |
| `nlb` | Network Load Balancer | Address type, DNS, VPC |
| `ess` | Auto Scaling (ESS) | Group type, active/max/min capacity |
| `ram` | RAM User | Display name, comments, created/updated time |
| `cms-ct` | CloudMonitor Alert Contact | Mail, SMS, channel verification state, groups |
| `cms-cg` | CloudMonitor Contact Group | Members, subscription, weekly report |
| `cms-alert` | CloudMonitor Alert Rule | Namespace, metric, thresholds (Critical/Warn/Info), instances, contact groups |
| `arms-ct` | ARMS Alert Contact | Name, email, phone, verification status |
| `arms-cg` | ARMS Contact Group | Name, contacts, status |
| `arms-alert` | ARMS Alert Rule | Name, status, region, level, type, notification ways |
| `cas` | Certificate (SSL/TLS) | Domain, issuer, expiration date, status, certificate ID |
| `ack` | Container Service (ACK) | Cluster ID, name, state, version, node count, region |
| `acr` | Container Registry (ACR) | Instance ID, name, specification, namespace count, repository count, ACL entries |
| `acl` | Access Control List | ACL ID, name, IP entries, bound ALBs, listener count |
| `kfk` | Message Queue for Kafka | Instance ID, name, series, topics, SASL users, consumer groups, ACLs, config params |
| `ck` | ClickHouse | Enterprise & community-compatible editions: architecture, node count, per-node/total storage, disk/CPU usage, endpoints, whitelist groups (ACL), accounts, backup policy |
| `prom` | Prometheus (ARMS) | Type, pay method, Grafana instance, storage/archive days, auth settings, remote write/read, push gateway, OpenTelemetry, HTTP API endpoints (public + VPC) |

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

- Provider-specific CLI tools (e.g. `aliyun` CLI for Alibaba Cloud, `hcloud` CLI for Huawei Cloud)
- A valid provider config (e.g. `~/.aliyun/config.json`, `~/.hcloud/config.json`).
  An alternate Aliyun config file can be used instead via `-c`/`--config`
  (see [Use an alternate CLI config file](#use-an-alternate-cli-config-file)).

## Installation

### Install from GitHub releases (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | bash
```

Custom install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | INSTALL_DIR=~/bin bash
```

### Install from GitHub releases (Windows PowerShell)

```powershell
Invoke-WebRequest -Uri https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.ps1 -OutFile install.ps1
.\install.ps1

# Custom directory
.\install.ps1 -InstallDir C:\tools
```

### Build from source

```bash
make install
```

Installs to `/usr/local/bin` when it is writable, otherwise falls back to
`~/.local/bin`. Override the destination with `make install PREFIX=...` or
`make install BINDIR=...`.

## CLI Usage

### Global flags

These flags apply to every command, including the TUI:

| Flag | Description |
|------|-------------|
| `-c, --config <path>` | Use an alternate vendor CLI config file. The vendor is auto-detected from the file contents and the run processes that vendor only (Aliyun only — `hcloud` has no custom config option) |
| `--profile <name>` | Override the active profile (defaults to the config file's `current` profile) |

Resource subcommands additionally accept `--region <id>` (filter by region)
and `--sync` (force re-sync before listing).

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

# CloudMonitor: alert rules with thresholds, contacts and contact groups
cloudres aliyun cms-alert
cloudres aliyun cms-ct --profile production
cloudres aliyun cms-cg --sync

# ARMS: alert contacts, contact groups, and alert rules
cloudres aliyun arms-ct
cloudres aliyun arms-cg
cloudres aliyun arms-alert

# SSL/TLS certificates
cloudres aliyun cas --sync

# Container Service (Kubernetes clusters)
cloudres aliyun ack

# Container Registry (instances with namespace/repo counts)
cloudres aliyun acr

# Access Control Lists (IP whitelist/blacklist for ALB)
cloudres aliyun acl --region cn-hangzhou

# Message Queue for Kafka (instances with topics, SASL users, ACLs)
cloudres aliyun kfk --profile l145 --sync

# ClickHouse (enterprise + community-compatible editions)
cloudres aliyun ck --profile blz-p-ntes-web --sync

# ARMS Prometheus instances (with API endpoints and storage duration)
cloudres aliyun prom --profile blz-p-ntes-web --sync

# Huawei Cloud
cloudres huawei ecs --sync
cloudres huawei eip
cloudres huawei dcs --profile myproject --sync
```

### Use an alternate CLI config file

The global `--config` flag points cloudres at a vendor CLI config file other
than the default (`~/.aliyun/config.json`, `~/.hcloud/config.json`). The vendor
is auto-detected from the file contents (profile key naming), and the run is
restricted to that vendor — TUI, `list`, and provider subcommands only see it.

```bash
# TUI restricted to aliyun, using the given config
cloudres -c /path/to/aliyun-config.json

# Same for CLI commands (--config is the long form)
cloudres -c /path/to/aliyun-config.json list
cloudres -c /path/to/aliyun-config.json aliyun ecs --sync
```

Notes:

- Only Aliyun is supported: the `aliyun` CLI accepts `--config-path`, while
  `hcloud` has no equivalent option (it always reads `~/.hcloud/config.json`),
  so a Huawei config passed to `--config` is rejected with an error.
- Data synced with an explicit config is cached in its own partition
  (`<profile>@<hash>`), so it never mixes with rows synced from the default
  config file.

### Profiles without a region

If an Aliyun profile has no `region_id`, region-scoped commands (`ecs`, `vpc`,
`vsw`, `rds`, `tair`, `pdb`, `slb`, `alb`, `nlb`, `ess`, `cas`, `ack`, `acl`,
`kfk`, `ck`, `prom`) do not query anything. Instead they print a warning that
includes the built-in region list and exit successfully; pass `--region` to
query a specific region:

```bash
cloudres aliyun ecs --region cn-hangzhou
```

Global resource types (`oss`, `ram`, `acr`, `cms-*`, `arms-*`) work without a
region.

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

### Region Select (`StateRegionSelect`)

Shown when an Aliyun profile has no `region_id`: pick a region to continue.

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate |
| `Enter` | Select region and enter the main view |
| `Esc` | Back to provider select |
| `q` | Quit |

### Main View (`StateMain`)

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate resources |
| `1` – `9` | Select region by index |
| `:` | Command mode — type a resource type (e.g. `ecs`, `subnet`) |
| `/` | Filter resources (case-insensitive substring search across all columns) |
| `d` | View resource detail |
| `Esc` | Go back (clear filter → clear resource type → region selection → provider select) |
| `q` | Quit |

### Detail View (`StateDetail`)

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Scroll detail content |
| `/` | Search within detail (filter key-value lines) |
| `Enter` | Confirm search |
| `Esc` / `d` | Back to main view (or clear search) |
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
