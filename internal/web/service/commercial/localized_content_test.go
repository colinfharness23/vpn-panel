package commercial

import "testing"

func TestValidLocalizedContentAllowsOneCompleteLocale(t *testing.T) {
	if !validLocalizedContent(`{"zh-CN":"服务公告"}`, `{"zh-CN":"欢迎使用 PHEERO。"}`) {
		t.Fatal("a complete Chinese translation should be editable without an English translation")
	}
	if !validLocalizedContent(`{"en-US":"Notice"}`, `{"en-US":"Welcome."}`) {
		t.Fatal("a complete English translation should be accepted")
	}
}

func TestValidLocalizedContentRejectsPartialOrEmptyLocales(t *testing.T) {
	tests := []struct {
		name     string
		titles   string
		contents string
	}{
		{name: "empty", titles: `{}`, contents: `{}`},
		{name: "title only", titles: `{"zh-CN":"服务公告"}`, contents: `{}`},
		{name: "content only", titles: `{}`, contents: `{"zh-CN":"正文"}`},
		{name: "mismatched locale", titles: `{"zh-CN":"服务公告"}`, contents: `{"en-US":"Body"}`},
		{name: "invalid json", titles: `{`, contents: `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if validLocalizedContent(test.titles, test.contents) {
				t.Fatal("invalid localized content was accepted")
			}
		})
	}
}
