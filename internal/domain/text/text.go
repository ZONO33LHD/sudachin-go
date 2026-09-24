package text

import (
	"unicode/utf8"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
)

// Text は書き換えを終えた解析用テキスト。生成後は変更されないため複数 goroutine から安全に読める。
type Text struct {
	original string
	modified string
	m2o      []int

	chars []rune
	cats  []chars.Category
	// catRun[i] は i 文字目から同じ文字種 (積集合が空でない) が続く文字数。
	catRun []int
	// c2b[i] は i 文字目の先頭バイト位置。末尾に len(modified) の番兵を持つ。
	c2b []int
	// b2c[i] は i バイト目を含む文字の位置。末尾に文字数の番兵を持つ。
	b2c []int
	// bow[i] は i バイト目から語を始めてよいか。
	bow []bool
}

// Build は Builder の現在の状態から解析用の Text を作る。
func (b *Builder) Build(cat chars.Categorizer) *Text {
	t := &Text{original: b.original, modified: b.current, m2o: b.m2o}
	n := utf8.RuneCountInString(t.modified)
	t.chars = make([]rune, 0, n)
	t.cats = make([]chars.Category, 0, n)
	t.c2b = make([]int, 0, n+1)
	t.b2c = make([]int, len(t.modified)+1)
	t.bow = make([]bool, len(t.modified))

	const nonStarting = chars.Alpha | chars.Greek | chars.Cyrillic
	var prev chars.Category
	nextBOW := true
	for off := 0; off < len(t.modified); {
		r, width := utf8.DecodeRuneInString(t.modified[off:])
		idx := len(t.chars)
		c := cat.Category(r)
		t.chars = append(t.chars, r)
		t.cats = append(t.cats, c)
		t.c2b = append(t.c2b, off)
		for i := range width {
			t.b2c[off+i] = idx
		}

		var canBOW bool
		switch {
		case !nextBOW:
			nextBOW, canBOW = true, false
		case c.Has(chars.NoOOVBOW2):
			nextBOW, canBOW = false, false
		case c.Has(chars.NoOOVBOW):
			canBOW = false
		case c.Has(nonStarting):
			canBOW = !c.Has(prev)
		default:
			canBOW = true
		}
		t.bow[off] = canBOW
		prev = c
		off += width
	}
	t.c2b = append(t.c2b, len(t.modified))
	t.b2c[len(t.modified)] = len(t.chars)

	t.catRun = make([]int, len(t.chars))
	var common chars.Category
	for i := len(t.cats) - 1; i >= 0; i-- {
		if i+1 < len(t.cats) && t.cats[i]&common&chars.All != 0 {
			common &= t.cats[i]
			t.catRun[i] = t.catRun[i+1] + 1
			continue
		}
		common = t.cats[i]
		t.catRun[i] = 1
	}
	return t
}

// Modified は正規化後のテキストを返す。
func (t *Text) Modified() string { return t.modified }

// Original は元のテキストを返す。
func (t *Text) Original() string { return t.original }

// CharLen は正規化後テキストの文字数を返す。
func (t *Text) CharLen() int { return len(t.chars) }

// Char は i 文字目を返す。
func (t *Text) Char(i int) rune { return t.chars[i] }

// Category は i 文字目の文字種を返す。
func (t *Text) Category(i int) chars.Category { return t.cats[i] }

// CategoryOfRange は [begin, end) 文字の文字種の積集合を返す。
func (t *Text) CategoryOfRange(begin, end int) chars.Category {
	if begin >= end {
		return 0
	}
	c := chars.Category(^uint32(0))
	for _, x := range t.cats[begin:end] {
		c &= x
	}
	return c
}

// CategoryRun は i 文字目から同じ文字種が続く文字数を返す。
func (t *Text) CategoryRun(i int) int { return t.catRun[i] }

// ByteOffset は i 文字目の先頭バイト位置を返す。i == CharLen() なら末尾を返す。
func (t *Text) ByteOffset(i int) int { return t.c2b[i] }

// CharIndex はバイト位置 b を含む文字の位置を返す。b == len(Modified()) なら文字数を返す。
func (t *Text) CharIndex(b int) int { return t.b2c[b] }

// CanBOW はバイト位置 b から語を始めてよいかを返す。
func (t *Text) CanBOW(b int) bool { return b < len(t.bow) && t.bow[b] }

// WordCandidateLength は i 文字目から次に語を始められる位置までの文字数を返す。
func (t *Text) WordCandidateLength(i int) int {
	for j := i + 1; j < len(t.chars); j++ {
		if t.bow[t.c2b[j]] {
			return j - i
		}
	}
	return len(t.chars) - i
}

// ModifiedSlice は正規化後テキストのバイト範囲 [begin, end) を返す。
func (t *Text) ModifiedSlice(begin, end int) string { return t.modified[begin:end] }

// OriginalRange は正規化後テキストのバイト範囲に対応する元テキストのバイト範囲を返す。
func (t *Text) OriginalRange(begin, end int) (int, int) { return t.m2o[begin], t.m2o[end] }

// OriginalSlice は正規化後テキストのバイト範囲に対応する元テキストを返す。
func (t *Text) OriginalSlice(begin, end int) string {
	ob, oe := t.OriginalRange(begin, end)
	return t.original[ob:oe]
}
