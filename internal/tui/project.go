package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/scaffold"
)

func (u *wizard) project() error {
	opts := scaffold.Options{Stdout: u.out, Stderr: u.errOut, Mirror: os.Getenv("LHCLI_MIRROR")}
	if err := u.form(huh.NewSelect[scaffold.Kind]().Title("选择项目模板").Options(
		huh.NewOption("Go 后端项目", scaffold.KindGo),
		huh.NewOption("Flutter 移动端项目", scaffold.KindMobile),
	).Value(&opts.Kind), huh.NewInput().Title("项目名称").Placeholder("例如 order-api 或 my_shop").Value(&opts.Name).
		Validate(func(value string) error { _, err := scaffold.NormalizeName(opts.Kind, value); return err })); err != nil {
		return err
	}
	name, err := scaffold.NormalizeName(opts.Kind, opts.Name)
	if err != nil {
		return err
	}
	opts.Name = name
	opts.Dir = "./" + name
	fields := []huh.Field{u.input("项目保存位置", &opts.Dir, validateProjectDir)}
	if opts.Kind == scaffold.KindGo {
		opts.Module = "github.com/example/" + name
		fields = append(fields, u.input("Go module 路径", &opts.Module, required).Description("可替换为自己的代码仓库地址"))
	} else {
		opts.Org = "com.example"
		opts.DisplayName = strings.ReplaceAll(name, "_", " ")
		fields = append(fields,
			u.input("应用组织标识", &opts.Org, required).Description("反向域名，例如 com.example"),
			u.input("应用显示名称", &opts.DisplayName, required))
	}
	gitInit, install := true, true
	fields = append(fields,
		huh.NewConfirm().Title("初始化 Git 仓库？").Affirmative("初始化").Negative("跳过").Value(&gitInit),
		huh.NewConfirm().Title("创建后安装项目依赖？").Affirmative("安装").Negative("跳过").Value(&install))
	if err := u.form(fields...); err != nil {
		return err
	}
	opts.Dir, err = expandPath(opts.Dir)
	if err != nil {
		return err
	}
	opts.NoGit, opts.NoInstall = !gitInit, !install
	fmt.Fprintf(u.out, "\n项目: %s\n模板: %s\n保存到: %s\n", name, opts.Kind, opts.Dir)
	if opts.Kind == scaffold.KindGo {
		fmt.Fprintf(u.out, "Go module: %s\n", opts.Module)
	} else {
		fmt.Fprintf(u.out, "组织标识: %s\n应用名称: %s\n", opts.Org, opts.DisplayName)
	}
	fmt.Fprintf(u.out, "初始化 Git: %s\n安装依赖: %s\n", yesNo(gitInit), yesNo(install))
	fmt.Fprintln(u.out, "将下载项目模板并执行模板中的改名脚本。")
	confirmed, err := u.confirm("确认创建这个项目？")
	if err != nil || !confirmed {
		return err
	}
	return scaffold.Run(u.ctx, opts)
}

func validateProjectDir(value string) error {
	if err := required(value); err != nil {
		return err
	}
	path, err := expandPath(value)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("保存位置已存在且不是普通目录")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("该目录非空，请选择新的目录；向导不会覆盖已有文件")
	}
	return nil
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
