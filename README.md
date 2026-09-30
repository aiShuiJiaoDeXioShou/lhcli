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

### 从模板初始化项目

```bash
# 后端（linghe-go-template）
lhcli init go order-api --module github.com/you/order-api

# 移动端（linghe_mobile_template）
lhcli init mobile my_shop --org com.you --display-name "我的商店"

# 预览改名结果，不写入磁盘
lhcli init go order-api --dry-run
```

`init` 会下载模板 tarball、执行模板内固定的 `scripts/rename.sh` 完成改名，
再初始化 Git 仓库并安装依赖。常用参数：

| 参数 | 说明 |
| --- | --- |
| `--dir` | 目标目录，默认 `./<name>` |
| `--force` | 目标目录非空时先删除 |
| `--no-git` | 跳过 Git 初始化 |
| `--no-install` | 跳过依赖安装 |
| `--dry-run` | 只预览，不写入 |
| `--template-repo` / `--template-ref` | 覆盖模板仓库与 ref |
| `--mirror` | 下载镜像前缀，例如 `https://gh-proxy.com/` |

`init go` 可用 `--module` 指定 Go module 路径（默认 `github.com/example/<name>`）；
`init mobile` 可用 `--org` 指定反向域名（默认 `com.example`）。

> Windows 下 `init` 需要 Git Bash 提供的 `sh`，lhcli 会自动探测 Git for Windows 的安装路径。

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
├── internal/scaffold/      # init 命令：模板下载、改名、初始化
├── .goreleaser.yaml        # 跨平台发布配置
├── .github/workflows/      # CI 与发布流水线
└── install.sh              # curl | sh 安装脚本
```

## 许可证

MIT，详见 [LICENSE](LICENSE)。
