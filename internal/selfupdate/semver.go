package selfupdate

import (
	"fmt"
	"strconv"
	"strings"
)

// version 表示一个解析后的语义化版本号。
type version struct {
	numbers    [3]int
	prerelease []string
}

// Compare 比较两个语义化版本号：a 小于 b 返回 -1，相等返回 0，大于返回 1。
func Compare(a, b string) (int, error) {
	left, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	right, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	return left.compare(right), nil
}

// IsValid 判断字符串是否为可解析的语义化版本号。
func IsValid(v string) bool {
	_, err := parseVersion(v)
	return err == nil
}

// parseVersion 解析形如 v1.2.3、1.2.3-rc.1 的版本号，忽略构建元数据。
func parseVersion(value string) (version, error) {
	original := value
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return version{}, fmt.Errorf("非法的版本号: %q", original)
	}

	// 构建元数据（+xxx）不参与比较。
	if index := strings.IndexByte(value, '+'); index >= 0 {
		value = value[:index]
	}

	core := value
	var prerelease []string
	if index := strings.IndexByte(value, '-'); index >= 0 {
		core = value[:index]
		prerelease = strings.Split(value[index+1:], ".")
	}

	parts := strings.Split(core, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return version{}, fmt.Errorf("非法的版本号: %q", original)
	}

	var parsed version
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return version{}, fmt.Errorf("非法的版本号: %q", original)
		}
		parsed.numbers[i] = number
	}
	parsed.prerelease = prerelease
	return parsed, nil
}

// compare 按 semver 2.0 规则比较两个版本。
func (v version) compare(other version) int {
	for i := range v.numbers {
		if v.numbers[i] != other.numbers[i] {
			if v.numbers[i] < other.numbers[i] {
				return -1
			}
			return 1
		}
	}

	// 无预发布段者版本更高。
	if len(v.prerelease) == 0 && len(other.prerelease) == 0 {
		return 0
	}
	if len(v.prerelease) == 0 {
		return 1
	}
	if len(other.prerelease) == 0 {
		return -1
	}
	return comparePrerelease(v.prerelease, other.prerelease)
}

// comparePrerelease 比较预发布段。
func comparePrerelease(left, right []string) int {
	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	for i := 0; i < length; i++ {
		if cmp := compareIdentifier(left[i], right[i]); cmp != 0 {
			return cmp
		}
	}
	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	default:
		return 0
	}
}

// compareIdentifier 比较单个预发布标识：纯数字按数值比较且优先级低于字母标识。
func compareIdentifier(left, right string) int {
	leftNumber, leftErr := strconv.Atoi(left)
	rightNumber, rightErr := strconv.Atoi(right)

	switch {
	case leftErr == nil && rightErr == nil:
		switch {
		case leftNumber < rightNumber:
			return -1
		case leftNumber > rightNumber:
			return 1
		default:
			return 0
		}
	case leftErr == nil:
		return -1
	case rightErr == nil:
		return 1
	default:
		return strings.Compare(left, right)
	}
}
