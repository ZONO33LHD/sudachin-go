package inputtext

import (
	"unicode/utf8"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// IgnoreYomigana は「漢字（よみがな）」のように漢字の直後に括弧で添えられた読み仮名を取り除く。
type IgnoreYomigana struct {
	left, right map[rune]bool
	maxLength   int
	isKanji     func(rune) bool
	isReading   func(rune) bool
}

var _ tokenize.InputTextPlugin = (*IgnoreYomigana)(nil)

// NewIgnoreYomigana は括弧の集合と読み仮名の最大文字数、文字の判定関数から作る。
func NewIgnoreYomigana(left, right []rune, maxLength int, isKanji, isReading func(rune) bool) *IgnoreYomigana {
	toSet := func(rs []rune) map[rune]bool {
		s := make(map[rune]bool, len(rs))
		for _, r := range rs {
			s[r] = true
		}
		return s
	}
	return &IgnoreYomigana{left: toSet(left), right: toSet(right), maxLength: maxLength, isKanji: isKanji, isReading: isReading}
}

// Rewrite は漢字直後の読み仮名 (括弧を含む) を削除する置換を e に登録する。
func (p *IgnoreYomigana) Rewrite(current string, e *text.Editor) error {
	for i := 0; i < len(current); {
		r, size := utf8.DecodeRuneInString(current[i:])
		i += size
		if !p.isKanji(r) {
			continue
		}
		if end, ok := p.matchYomigana(current, i); ok {
			e.Replace(i, end, "")
			i = end
		}
	}
	return nil
}

// matchYomigana は start から「左括弧 + 読み仮名 1〜maxLength 文字 + 右括弧」が続くとき、その終端を返す。
func (p *IgnoreYomigana) matchYomigana(s string, start int) (int, bool) {
	r, size := utf8.DecodeRuneInString(s[start:])
	if !p.left[r] {
		return 0, false
	}
	i, n := start+size, 0
	for i < len(s) {
		r, size = utf8.DecodeRuneInString(s[i:])
		if !p.isReading(r) {
			break
		}
		i += size
		n++
	}
	if n == 0 || n > p.maxLength || i >= len(s) {
		return 0, false
	}
	if r, size = utf8.DecodeRuneInString(s[i:]); !p.right[r] {
		return 0, false
	}
	return i + size, true
}
