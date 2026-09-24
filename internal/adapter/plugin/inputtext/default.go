// Package inputtext は解析前に入力テキストを書き換えるプラグインを実装する。
package inputtext

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// Normalizer は Sudachi の DefaultInputTextPlugin 相当の正規化を行う。
//
//  1. 置換規則 (rewrite.def の 2 列の行) に一致する最長の文字列を置き換える。
//  2. それ以外の文字は小文字化し、正規化除外文字でなければ NFKC 正規化も行う。
type Normalizer struct {
	ignore  map[rune]bool
	replace map[string]string
	// maxKeyLen は置換規則のキーの、先頭文字ごとの最大バイト長。
	maxKeyLen map[rune]int
}

var _ tokenize.InputTextPlugin = (*Normalizer)(nil)

// NewNormalizer は正規化除外文字 ignore と置換規則 replace から Normalizer を作る。
func NewNormalizer(ignore map[rune]bool, replace map[string]string) *Normalizer {
	n := &Normalizer{ignore: ignore, replace: replace, maxKeyLen: map[rune]int{}}
	for k := range replace {
		first, _ := utf8.DecodeRuneInString(k)
		n.maxKeyLen[first] = max(n.maxKeyLen[first], len(k))
	}
	return n
}

// Rewrite は current を正規化する置換を e に登録する。
func (n *Normalizer) Rewrite(current string, e *text.Editor) error {
	for i := 0; i < len(current); {
		r, size := utf8.DecodeRuneInString(current[i:])
		if key, with, ok := n.longestReplacement(current[i:], r); ok {
			e.Replace(i, i+len(key), with)
			i += len(key)
			continue
		}
		if s, changed := n.normalizeRune(r); changed {
			e.Replace(i, i+size, s)
		}
		i += size
	}
	return nil
}

func (n *Normalizer) longestReplacement(s string, first rune) (key, with string, ok bool) {
	limit, found := n.maxKeyLen[first]
	if !found {
		return "", "", false
	}
	for l := min(limit, len(s)); l > 0; l-- {
		if with, ok := n.replace[s[:l]]; ok {
			return s[:l], with, true
		}
	}
	return "", "", false
}

func (n *Normalizer) normalizeRune(r rune) (string, bool) {
	s := string(r)
	needNFKC := !n.ignore[r] && !norm.NFKC.IsNormalString(s)
	if isUpper(r) {
		s = strings.ToLower(s)
	}
	if needNFKC {
		s = norm.NFKC.String(s)
	}
	return s, s != string(r)
}

// isUpper は Unicode の Uppercase 属性を判定する。unicode.IsUpper (一般カテゴリ Lu) だけでは
// ローマ数字「Ⅰ」や丸囲み文字「Ⓐ」などの Other_Uppercase を取りこぼす。
func isUpper(r rune) bool {
	return unicode.IsUpper(r) || unicode.Is(unicode.Other_Uppercase, r)
}
