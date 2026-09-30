// Command lhcli 是一个使用 Go 编写的自定义命令行工具。
package main

import (
	"github.com/aiShuiJiaoDeXioShou/lhcli/cmd"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/selfupdate"
)

func main() {
	// 清理上次自更新遗留的备份文件，失败时忽略。
	selfupdate.CleanupStale()
	cmd.Execute()
}
