# lhcli

A custom command-line tool written in Go.

## Installation

### 1. Go install (requires Go toolchain)

```bash
go install github.com/aiShuiJiaoDeXioShou/lhcli@latest
```

### 2. Install script (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/aiShuiJiaoDeXioShou/lhcli/main/install.sh | sh
```

Or pin a version / change the destination:

```bash
VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" sh install.sh
```

### 3. Download a release binary

Grab the archive for your OS/arch from the
[releases page](https://github.com/aiShuiJiaoDeXioShou/lhcli/releases)
and put the `lhcli` binary somewhere on your `PATH`.

## Usage

```bash
lhcli --help
lhcli version
lhcli greet --name world
```

## Development

```bash
go mod tidy
go build -o lhcli .
go test ./...
```

Build with version metadata injected:

```bash
go build -ldflags "\
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Version=v0.1.0 \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o lhcli .
```

## Releasing

Tag and push; the release workflow runs GoReleaser and publishes
cross-platform binaries plus `checksums.txt`.

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Project layout

```
.
├── main.go                 # thin entry point
├── cmd/                    # cobra commands
├── internal/buildinfo/     # build-time metadata (ldflags target)
├── .goreleaser.yaml        # cross-platform release config
├── .github/workflows/      # CI + release pipelines
└── install.sh              # curl | sh installer
```

## License

MIT - see [LICENSE](LICENSE).
