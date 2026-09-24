package inputtext

import (
	"testing"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// 大文字判定 (標準ライブラリ unicode) と NFKC 正規化 (x/text) の Unicode の版が違うと、
// 新しく追加された文字だけ片方の処理しか効かなくなる。Go 1.27 と x/text v0.42 はどちらも Unicode 17。
func TestUnicodeVersionsMatch(t *testing.T) {
	t.Parallel()
	if unicode.Version != norm.Version {
		t.Errorf("unicode.Version = %s but norm.Version = %s; align the Go toolchain and golang.org/x/text", unicode.Version, norm.Version)
	}
}
