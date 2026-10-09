# lhcli

使用 Go 编写的个人开发工具，提供项目模板初始化、多 agent 技能管理和自更新。

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
lhcli update --check
lhcli skills list --global
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

### 管理个人技能

先登记已有的 Git 技能仓库，再将 `skills/<名称>/` 链接到 agent 的技能目录：

```bash
lhcli skills init --path /path/to/my-skills
lhcli skills list --global
lhcli skills enable go-api --agent codex,claude --global
lhcli skills update --check
lhcli skills disable go-api --agent claude --global
```

支持 Codex、Claude Code 和 Cursor。`--global` 操作用户目录，省略时操作当前目录。
也可用 `init --repo owner/my-skills` 克隆到 `~/.lhcli/repos/my-skills/`。
同名的手工目录不会被覆盖；更新要求工作区干净且只允许快进。
Windows 需要创建符号链接的权限。完整目录约定、批量操作和恢复说明见[技能管理文档](docs/skills.md)。

### 命令帮助与补全

```bash
lhcli help skills
lhcli skills enable --help
lhcli completion zsh
```

`completion` 支持 `bash`、`zsh`、`fish`、`powershell`，输出脚本到标准输出。
`greet` 示例命令仍可使用，但不再出现在主帮助菜单中。

### 更新 lhcli 自身

```bash
# 检查是否有新版本（不下载）
lhcli update --check

# 更新到最新版本，交互确认后替换当前二进制
lhcli update

# 直接指定版本、跳过确认或预览动作
lhcli update --version v0.3.0 --yes
lhcli update --dry-run
```

更新流程会从 GitHub Releases 下载对应平台的压缩包，
用发布目录中的 `checksums.txt` 校验 SHA256 后再替换当前可执行文件。

| 参数 | 说明 |
| --- | --- |
| `--check` | 只检查是否有新版本 |
| `--version` | 安装指定版本，默认最新版 |
| `--force` | 已是最新或本地为 `dev` 时也强制重装 |
| `--yes` / `-y` | 跳过交互确认（非交互式终端下必须提供） |
| `--dry-run` | 只打印将要执行的动作 |
| `--mirror` | 下载镜像前缀，例如 `https://gh-proxy.com/`（默认读取环境变量 `LHCLI_MIRROR`，命令行优先） |
| `--repo` | 覆盖发布仓库 owner/name |

> `go build` 得到的本地 `dev` 版本需加 `--force` 才会被覆盖；
> 匿名调用 GitHub API 有每小时限额，必要时设置 `GITHUB_TOKEN` 或 `GH_TOKEN`。
> Windows 下旧版本会备份为二进制旁的 `.old` 文件，并在下次启动时自动清理。

## 开发

```bash
go mod tidy
go vet ./...
go build -o lhcli .
go test ./...
sh scripts/test-install.sh
```

安装脚本自检适用于 Linux / macOS，使用本地下载替身，不访问网络或系统安装目录。
协作约定见 [AGENTS.md](AGENTS.md)。

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
├── internal/selfupdate/    # update 命令：版本检查、下载校验、替换二进制
├── internal/skills/        # skills 命令：仓库登记、更新与多 agent 链接
├── docs/skills.md          # 技能使用说明与恢复方法
├── scripts/               # 安装脚本自检
├── .goreleaser.yaml        # 跨平台发布配置
├── .github/workflows/      # CI 与发布流水线
└── install.sh              # curl | sh 安装脚本
```

## 许可证

MIT，详见 [LICENSE](LICENSE)。
