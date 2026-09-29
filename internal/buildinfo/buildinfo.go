// Package buildinfo 保存通过 -ldflags 在构建期注入的元数据。
package buildinfo

// 以下变量的值会在构建时被覆盖，例如：
//
//	go build -ldflags "-X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Version=v1.0.0"
var (
	// Version 是本次构建的语义化版本号。
	Version = "dev"
	// Commit 是构建该二进制所对应的 git 提交。
	Commit = "none"
	// Date 是构建时间戳。
	Date = "unknown"
)
