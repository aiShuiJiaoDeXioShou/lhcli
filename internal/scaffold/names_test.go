package scaffold

import "testing"

// TestNormalizeName 覆盖项目名规范化的主要场景。
func TestNormalizeName(t *testing.T) {
	cases := []struct {
		name    string
		kind    Kind
		input   string
		want    string
		wantErr bool
	}{
		{name: "go 空格转短横线", kind: KindGo, input: "Order Service", want: "order-service"},
		{name: "go 下划线转短横线", kind: KindGo, input: "order_service", want: "order-service"},
		{name: "go 连续分隔符折叠", kind: KindGo, input: "Order__Service", want: "order-service"},
		{name: "go 数字开头报错", kind: KindGo, input: "9order", wantErr: true},
		{name: "go 空输入报错", kind: KindGo, input: "   ", wantErr: true},
		{name: "mobile 空格转下划线", kind: KindMobile, input: "My Shop", want: "my_shop"},
		{name: "mobile 短横线转下划线", kind: KindMobile, input: "my-shop", want: "my_shop"},
		{name: "mobile 保留数字", kind: KindMobile, input: "app2", want: "app2"},
		{name: "mobile 数字开头报错", kind: KindMobile, input: "123app", wantErr: true},
		{name: "mobile 全部为符号报错", kind: KindMobile, input: "--- ", wantErr: true},
		{name: "mobile 前导分隔符被丢弃", kind: KindMobile, input: "-app", want: "app"},
		{name: "未知类型报错", kind: Kind("worker"), input: "app", wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := NormalizeName(testCase.kind, testCase.input)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际得到 %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("未期望报错: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("期望 %q，实际 %q", testCase.want, got)
			}
		})
	}
}
