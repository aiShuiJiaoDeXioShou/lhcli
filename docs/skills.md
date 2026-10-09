# 个人技能管理

`lhcli skills` 管理一个个人 Git 技能仓库，将选中的技能链接到 Codex、Claude Code 和 Cursor。修改源码后，链接读取同一份文件；正在运行的 agent 是否立即刷新，由对应工具决定。

## 准备仓库

技能源码建议放在独立 Git 仓库中，可以使用私有仓库。仓库结构如下：

```text
my-skills/
├── .git/
└── skills/
    ├── go-api/
    │   └── SKILL.md
    └── code-review/
        ├── SKILL.md
        └── references/
```

每个技能是 `skills/` 的直接子目录。目录名最多 64 个字符，仅使用小写英文字母、数字及单个连字符，不能以连字符开头或结尾。`SKILL.md` 必须是非空普通文件。技能内部允许普通文件和子目录，不允许符号链接、子模块或特殊文件。

lhcli 检查文件结构，不解析或改写 YAML 元数据。请按照 [Agent Skills 规范](https://agentskills.io/specification) 编写 `name`、`description` 和正文；涉及工具专属能力时，还需在该工具中验证。

## 首次设置

已有本地仓库时，直接登记其根目录：

```bash
lhcli skills init --path /path/to/my-skills
```

也可以克隆远端仓库，下面的 `owner/my-skills` 需替换为真实仓库：

```bash
lhcli skills init --repo owner/my-skills
lhcli skills init --repo git@github.com:owner/my-skills.git
```

`--repo` 和 `--path` 必须且只能提供一个。克隆目录默认为 `~/.lhcli/repos/my-skills/`，配置保存在 `~/.lhcli/skills.json`。本机绝对路径和链接记录不写进技能仓库。

需要系统安装 Git。私有仓库使用本机已配置的 Git 凭据或 SSH；lhcli 不维护独立的认证配置，也不自动运行 `gh auth login`。Git 操作关闭终端凭据提问；认证失败时，先在终端配置 Git 认证再重试。

已有配置不会被新的仓库替换。重复登记同一个本地路径会直接成功。默认克隆目录已经存在时，使用 `--path` 登记，不覆盖目录。

## 查看、启用与停用

```bash
# 查看用户级技能，以及三个 agent 目录的链接状态
lhcli skills list --global

# 为两个 agent 启用一个技能
lhcli skills enable go-api --agent codex,claude --global

# 为当前目录启用技能；不会向上查找 Git 根目录
lhcli skills enable go-api --agent codex

# 先预览，再启用当前仓库的全部技能
lhcli skills enable --all --agent codex --global --dry-run
lhcli skills enable --all --agent codex --global

# 仅移除 Claude Code 的受管链接，保留源码
lhcli skills disable go-api --agent claude --global
```

| 参数 | 行为 |
| --- | --- |
| `--agent` | 选择 `codex`、`claude`、`cursor`，支持逗号分隔；启用和停用时必填 |
| `--global` | 操作用户目录；不提供时操作当前目录 |
| `--all` | 启用时选择当前仓库所有技能；停用时选择指定作用域和 agent 的所有受管链接 |
| `--dry-run` | 预览操作，不写入配置或链接 |

每次只链接单个技能目录，不替换整个 agent 技能目录。重复启用不会产生重复记录；受管链接丢失时，再次启用会修复。`--all` 不会自动启用未来新增的技能，新增后重新执行即可。

| Agent | 用户级目录 | 当前目录下的位置 |
| --- | --- | --- |
| Codex | `~/.agents/skills/` | `.agents/skills/` |
| Claude Code | `~/.claude/skills/` | `.claude/skills/` |
| Cursor | `~/.cursor/skills/` | `.cursor/skills/` |

这些是 lhcli 的写入位置，不是 agent 的全部扫描位置。Cursor 等工具可能同时读取其他工具的目录；`--agent` 不提供严格的技能可见性隔离。多个目录可能指向同一技能。

`list` 显示已启用、未启用、未受管、链接缺失、源码缺失或冲突。列表只表示当前目录或用户目录中的安装状态，不代表 agent 的实时技能目录。

## 更新与编辑

```bash
# 获取远端引用，检查更新；不修改工作区和 HEAD
lhcli skills update --check

# 更新当前分支到上游，只允许快进
lhcli skills update

# 仅显示计划，不联网、不获取远端引用
lhcli skills update --dry-run
```

更新使用当前分支配置的 upstream。未提交修改、未跟踪文件、游离 HEAD、没有 upstream 或分叉历史都会阻止自动更新。本地领先上游时保留本地提交，不回退版本。更新不会覆盖被忽略的本地文件，不执行 Git hooks。

获取更新后先检查上游技能结构。如果上游移除了任意已启用技能，先停用该技能在所有已登记作用域中的链接，再更新。

日常编辑直接在本地技能仓库中进行，用 Git 提交和推送。更新仓库不会执行技能脚本，也不会安装技能依赖。链接模式没有每个 agent 独立的版本：修改一份源码会影响所有指向它的链接。

## 冲突与恢复

- 同名普通目录、文件和未登记的手工链接不会被接管、覆盖或删除，即使手工链接指向同一份源码。
- 已登记的链接被改成其他内容后，操作会报冲突；请先自行检查该路径。
- agent 的配置目录或 `skills` 目录本身是符号链接时，操作会拒绝，避免越界写入或混淆所有权。
- 批量启用和停用先检查全部目标，执行中失败会尽力恢复本批次已经改动的链接。恢复失败会列出对应路径；新建的空父目录可能保留。
- 进程崩溃或断电不保证跨多个文件的事务完整性。先用 `list` 检查；未登记的残留链接需手工核实后处理，lhcli 不自动接管。
- 写操作使用 `~/.lhcli/skills.lock` 防止多个 lhcli 进程同时改动配置。异常退出留下锁文件时，确认没有运行中的 skills 命令后再删除锁文件。
- 不要移动或删除已经登记的本地仓库。确需迁移时，先停用所有链接、备份配置并移走 `skills.json`，再登记新路径和重新启用。

Windows 需要开发者模式或创建符号链接的权限。创建失败会明确报错，不会静默改为副本。项目级链接使用本机绝对路径，不适合提交给其他电脑使用；可以通过 `.git/info/exclude` 忽略这些链接。云端 agent、容器和远程机器需要在其运行环境中单独安装。
