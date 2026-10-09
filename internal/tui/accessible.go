package tui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"charm.land/huh/v2"
	"github.com/muesli/cancelreader"
)

type inputEnd struct{ err error }

type lineInput struct {
	r      *bufio.Reader
	length int
}

func (r *lineInput) Read(p []byte) (int, error) {
	for i := range p {
		b, err := r.r.ReadByte()
		if err != nil {
			// Huh 的逐项提示不传播 EOF；必须立即中止，避免默认值被当作用户确认。
			panic(inputEnd{err})
		}
		r.length++
		if r.length > 8192 {
			panic(inputEnd{fmt.Errorf("输入过长，请将每行限制在 8192 字节以内")})
		}
		p[i] = b
		if b == '\n' {
			r.length = 0
			return i + 1, nil
		}
	}
	return len(p), nil
}

func (u *wizard) runAccessible(start string) (err error) {
	reader, err := cancelreader.NewReader(u.in)
	if err != nil {
		return err
	}
	defer reader.Close()
	stop := context.AfterFunc(u.ctx, func() { reader.Cancel() })
	defer stop()
	u.in = &lineInput{r: bufio.NewReader(reader)}
	defer func() {
		if recovered := recover(); recovered != nil {
			end, ok := recovered.(inputEnd)
			if !ok {
				panic(recovered)
			}
			err = end.err
			if errors.Is(err, io.EOF) || errors.Is(err, cancelreader.ErrCanceled) {
				err = huh.ErrUserAborted
			}
		}
	}()
	return u.run(start)
}

var promptTranslations = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`Enter a number between (\d+) and (\d+):`), "请输入 $1 到 $2 之间的编号："},
	{regexp.MustCompile(`Select up to (\d+) options\.`), "最多选择 $1 项；输入编号切换勾选，输入 0 完成。"},
	{regexp.MustCompile(`Invalid: must be a number between (\d+) and (\d+)`), "请输入 $1 到 $2 之间的编号"},
}

type promptWriter struct{ io.Writer }

func (w promptWriter) Write(p []byte) (int, error) {
	value := string(p)
	for _, translation := range promptTranslations {
		value = translation.pattern.ReplaceAllString(value, translation.replacement)
	}
	value = strings.ReplaceAll(value, "There is only one option available; enter the number 1:", "只有一个选项，请输入 1：")
	value = strings.ReplaceAll(value, "invalid input. please try again", "输入无效，请输入 y 或 n")
	if _, err := io.WriteString(w.Writer, value); err != nil {
		return 0, err
	}
	return len(p), nil
}
