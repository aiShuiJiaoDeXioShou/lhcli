# lhcli

使用 Go 编写的自定义命令行工具。

## 安装

### 1. 通过 go install（需要 Go 工具链）

```bash
go install github.com/aiShuiJiaoDeXioShou/lhcli@latest
```

### 2. 通过安装脚本（Linux / macOS）

```bash
curl -fsSL https://raw.githubusercontent.com/aiShuiJiaoDeXioShou/lhcli/main/install.sh | sh
```

也可以指定版本与安装目录：

```bash
VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" sh install.sh
```

### 3. 直接下载发布二进制

在 [Releases 页面](https://github.com/aiShuiJiaoDeXioShou/lhcli/releases) 下载对应系统的压缩包，
将 `lhcli` 二进制放入 `PATH` 即可。

## 使用

```bash
lhcli --help
lhcli version
lhcli greet --name 世界
```

## 开发

```bash
go mod tidy
go build -o lhcli .
go test ./...
```

带版本信息构建：

```bash
go build -ldflags "\
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Version=v0.1.0 \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o lhcli .
```

## 发布

打 tag 并推送后，发布工作流会运行 GoReleaser，产出全平台二进制与 `checksums.txt`。

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 目录结构

```
.
├── main.go                 # 程序入口
├── cmd/                    # cobra 命令
├── internal/buildinfo/     # 构建期元数据（ldflags 注入点）
├── .goreleaser.yaml        # 跨平台发布配置
├── .github/workflows/      # CI 与发布流水线
└── install.sh              # curl | sh 安装脚本
```

## 许可证

MIT，详见 [LICENSE](LICENSE)。
