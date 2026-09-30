// Package scaffold 负责从 Git 模板仓库初始化新项目。
package scaffold

import (
	"fmt"
	"strings"
)

// Kind 表示模板类型。
type Kind string

const (
	// KindGo 表示 Go 后端模板。
	KindGo Kind = "go"
	// KindMobile 表示 Flutter 移动端模板。
	KindMobile Kind = "mobile"
)

// NormalizeName 依据模板类型把用户输入规范化为合法的项目名。
//
// go 使用短横线命名（order-service），mobile 使用下划线命名（my_shop）。
func NormalizeName(kind Kind, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("项目名不能为空")
	}

	var name string
	switch kind {
	case KindGo:
		name = normalizeSeparated(input, '-')
	case KindMobile:
		name = normalizeSeparated(input, '_')
	default:
		return "", fmt.Errorf("未知模板类型: %s", kind)
	}

	if name == "" {
		return "", fmt.Errorf("项目名不包含可用字符: %q", input)
	}
	if name[0] < 'a' || name[0] > 'z' {
		return "", fmt.Errorf("项目名必须以字母开头: %q", input)
	}
	return name, nil
}

// normalizeSeparated 把输入转为小写，并把非字母数字字符折叠为指定分隔符。
func normalizeSeparated(input string, separator byte) string {
	var builder strings.Builder
	pendingSeparator := false
	for _, r := range strings.ToLower(input) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pendingSeparator && builder.Len() > 0 {
				builder.WriteByte(separator)
			}
			pendingSeparator = false
			builder.WriteRune(r)
		default:
			pendingSeparator = true
		}
	}
	return builder.String()
}
