package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSkillsArguments(t *testing.T) {
	for _, args := range [][]string{
		{"skills", "enable"},
		{"skills", "enable", "go-api", "--all", "--agent", "codex"},
		{"skills", "init", "unexpected"},
		{"skills", "list", "unexpected"},
		{"skills", "update", "unexpected"},
	} {
		root := &cobra.Command{Use: "lhcli", SilenceErrors: true, SilenceUsage: true}
		root.AddCommand(newSkillsCmd())
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Fatalf("参数校验应失败: %v", args)
		}
	}
}

func TestSkillsHelp(t *testing.T) {
	root := &cobra.Command{Use: "lhcli"}
	root.AddCommand(newSkillsCmd())
	localizeHelp(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"skills", "enable", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--agent", "--global", "--dry-run", "codex,claude", "查看此命令的帮助"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("帮助中缺少 %s: %s", want, out.String())
		}
	}
}

func TestCompletion(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		for _, noDescriptions := range []bool{false, true} {
			root := &cobra.Command{Use: "lhcli"}
			root.CompletionOptions.DisableDefaultCmd = true
			root.AddCommand(newCompletionCmd(), newSkillsCmd())
			var out bytes.Buffer
			root.SetOut(&out)
			args := []string{"completion", shell}
			if noDescriptions {
				args = append(args, "--no-descriptions")
			}
			root.SetArgs(args)
			if err := root.Execute(); err != nil || out.Len() == 0 {
				t.Fatalf("生成 %s 补全失败: %v", shell, err)
			}
		}
	}
}
