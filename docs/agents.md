# AGENTS.md 指令模板管理

运行 `lhcli agents` 打开交互菜单：选择模板应用到项目、收录项目的指令、查看内容，以及提交和同步仓库。也可从 `lhcli` 主菜单进入。

指令模板与技能共用 `lhcli skills info` 显示的默认仓库。创建个人仓库时会生成一份通用 `default` 模板，可按自己的习惯编辑：

```text
my-skills/
├── skills/
└── agents/
    ├── default/
    │   └── AGENTS.md
    └── go-project/
        └── AGENTS.md
```

模板名使用小写字母、数字和连字符，最多 64 字符。文件必须为非空普通文件；不接受符号链接。这里只管理项目目录中的 `AGENTS.md`，不自动修改工具的全局指令，也不转换为 `CLAUDE.md` 或其他格式。

## 查看、应用与收录

```bash
# 查看模板列表与内容
lhcli agents list
lhcli agents show default

# 应用到当前目录，默认拒绝替换不同内容的已有文件
lhcli agents apply default

# 指定项目，先预览，再备份并替换
lhcli agents apply default --project ~/code/my-project --replace --dry-run
lhcli agents apply default --project ~/code/my-project --replace

# 把项目现有的 AGENTS.md 保存为新模板
lhcli agents capture go-project --project ~/code/my-project

# 把项目修改收回同名模板，先备份旧模板
lhcli agents capture go-project --project ~/code/my-project --replace
```

项目目录必须已存在，默认使用当前目录，不向上寻找项目根目录。应用的是完整文件的独立副本；不会自动合并已有指令，也不会在更新仓库时自动修改已应用的项目文件。项目修改可用 `capture` 收回；模板修改后可再次 `apply`。

交互流程先显示模板内容与复制方向，明确提示替换和备份，确认后执行。命令行必须提供 `--replace` 才能替换已有的不同内容；内容相同时直接返回。备份保存在 `~/.lhcli/backups/AGENTS-*.md`，每次使用独立名称，并在输出中显示准确路径。恢复时先核对内容，再把备份复制回提示中的目标路径。备份不会推送到 GitHub。

## 提交与同步

```bash
# 提交向导：同时包含 skills/ 和 agents/ 下的变更
lhcli agents submit

# 更新默认仓库，保持项目中已经应用的副本不变
lhcli skills update
```

`lhcli agents submit` 与 `lhcli skills submit` 使用同一套提交流程，不自动提交仓库的其他文件。推送会包含当前分支所有未推送的提交。已有暂存内容、上游配置、预览变化和推送失败的处理方式见 [技能管理说明](skills.md#提交和推送技能)。

指令内容由用户维护。工具只检查文件和路径，不保证各 agent 都会读取目标文件；在需要生效的工具和项目中检查其实际读取结果。
