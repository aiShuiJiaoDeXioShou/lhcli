# AGENTS.md

本文件是本仓库的协作约定，所有在此仓库工作的自动化代理与贡献者都应遵守。

## 语言约定（强制）

- **所有文档一律使用中文**：包括但不限于 `README.md`、`AGENTS.md`、`docs/` 下的文件、CHANGELOG 以及任何说明类 `.md` / `.txt` 文件。
- **所有代码注释一律使用中文**：包括 Go 源码的包注释、函数/类型/变量/常量注释、行内注释，以及 YAML 等配置文件中的注释。
- **面向用户的提示信息使用中文**：命令行输出、错误信息、安装脚本提示等。
- **以下内容保持英文，不属于翻译范围**：
  - `LICENSE` 等具有法律效力的文本；
  - 代码标识符（包名、函数名、变量名、类型名）、命令名、flag 名；
  - 第三方生成的文件及其文件头；
  - 无法翻译的专有名词（如 `GoReleaser`、`cobra`、`ldflags`）。

### 注释书写规范

为兼容 `go doc` / `golint`，Go 注释应以被注释对象名开头，正文使用中文：

```go
// Version 是本次构建的语义化版本号。
Version = "dev"

// Execute 执行根命令，出错时以非零状态码退出。
func Execute() { ... }
```

## 项目概览

- 语言：Go（go 1.27.1）
- 名称：`lhcli`
- 用途：自定义命令行工具，支持通过 git（`go install`）、安装脚本（`curl | sh`）、发布二进制三种方式安装
- 模块路径：`github.com/aiShuiJiaoDeXioShou/lhcli`

### 目录结构

- `main.go`：程序入口，仅调用 `cmd.Execute()`
- `cmd/`：基于 cobra 的命令定义
- `internal/`：内部实现，不可被外部包导入
- `internal/buildinfo/`：构建期元数据（`-ldflags` 注入点）
- `internal/scaffold/`：`init` 命令的模板下载、改名与初始化逻辑

## 常用命令

```bash
go mod tidy
go vet ./...
go test ./...
go build -o lhcli.exe .   # Windows 下产物带 .exe 后缀
```

## 构建与版本注入

版本信息通过 `-ldflags` 注入 `internal/buildinfo`：

```bash
go build -ldflags "\
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Version=v0.1.0 \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o lhcli .
```

## 提交与发布约定

- 提交信息使用「`type: 中文描述`」的约定式格式，例如 `feat: 新增 upgrade 子命令`。
- 发布通过打 tag 触发：`git tag v0.1.0 && git push origin v0.1.0`，随后由 GitHub Actions 运行 GoReleaser。
- 换行符由 `.gitattributes` 统一为 LF，请勿在 `install.sh` 等脚本中引入 CRLF。

## 环境说明

- 本机 `proxy.golang.org` 不可达，已设置 `GOPROXY=https://goproxy.cn,direct`。
- 到 `github.com` 的 HTTPS 推送偶发连接重置，失败时重试即可。

## 交付前自检

- [ ] 新增或修改的注释、文档、用户提示均为中文
- [ ] `go vet ./...` 与 `go build ./...` 通过
- [ ] 未提交构建产物（`lhcli.exe`、`dist/` 已在 `.gitignore` 中忽略）
