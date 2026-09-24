package text

import (
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
)

type categorizerFunc func(rune) chars.Category

func (f categorizerFunc) Category(r rune) chars.Category { return f(r) }

var simpleCategories = categorizerFunc(func(r rune) chars.Category {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return chars.Alpha
	case r >= '0' && r <= '9':
		return chars.Numeric
	case r == 'ー':
		return chars.Katakana | chars.NoOOVBOW
	case r >= 'ァ' && r <= 'ヶ':
		return chars.Katakana
	default:
		return chars.Default
	}
})

func TestRewriteKeepsOriginalMapping(t *testing.T) {
	t.Parallel()
	b := NewBuilder("ＡＢ漢字（かんじ）です")
	// 全角英字を半角小文字へ (3 バイト → 1 バイト)
	if err := b.Rewrite(func(cur string, e *Editor) error {
		e.Replace(0, 3, "a")
		e.Replace(3, 6, "b")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// 読み仮名を削除 (後続のプラグインは書き換え後の位置で置換を登録する)
	if err := b.Rewrite(func(cur string, e *Editor) error {
		if cur != "ab漢字（かんじ）です" {
			t.Fatalf("current = %q", cur)
		}
		e.Replace(8, 8+3*5, "")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	tx := b.Build(simpleCategories)

	if got, want := tx.Modified(), "ab漢字です"; got != want {
		t.Fatalf("Modified() = %q, want %q", got, want)
	}
	tests := []struct {
		begin, end int
		want       string
	}{
		{0, 1, "Ａ"},
		{1, 2, "Ｂ"},
		{2, 8, "漢字（かんじ）"},
		{8, 14, "です"},
		{0, len(tx.Modified()), "ＡＢ漢字（かんじ）です"},
	}
	for _, tt := range tests {
		if got := tx.OriginalSlice(tt.begin, tt.end); got != tt.want {
			t.Errorf("OriginalSlice(%d, %d) = %q, want %q", tt.begin, tt.end, got, tt.want)
		}
	}
}

func TestRewriteRejectsOverlappingEdits(t *testing.T) {
	t.Parallel()
	b := NewBuilder("abcdef")
	err := b.Rewrite(func(_ string, e *Editor) error {
		e.Replace(0, 3, "x")
		e.Replace(2, 4, "y")
		return nil
	})
	if err == nil {
		t.Fatal("overlapping edits should be rejected")
	}
	if b.Current() != "abcdef" {
		t.Errorf("a rejected rewrite must not change the text, got %q", b.Current())
	}
}

func TestBuildIndexes(t *testing.T) {
	t.Parallel()
	tx := NewBuilder("abアー1").Build(simpleCategories)

	if got := tx.CharLen(); got != 5 {
		t.Fatalf("CharLen() = %d, want 5", got)
	}
	wantOffsets := []int{0, 1, 2, 5, 8, 9}
	for i, want := range wantOffsets {
		if got := tx.ByteOffset(i); got != want {
			t.Errorf("ByteOffset(%d) = %d, want %d", i, got, want)
		}
	}
	if got := tx.CharIndex(3); got != 2 {
		t.Errorf("CharIndex(3) (inside 'ア') = %d, want 2", got)
	}
	// 英字の途中や NOOOVBOW の文字からは語を始められない
	wantBOW := map[int]bool{0: true, 1: false, 2: true, 5: false, 8: true}
	for b, want := range wantBOW {
		if got := tx.CanBOW(b); got != want {
			t.Errorf("CanBOW(%d) = %v, want %v", b, got, want)
		}
	}
	if got := tx.CategoryRun(2); got != 2 {
		t.Errorf("CategoryRun(2) (アー) = %d, want 2", got)
	}
	if got := tx.WordCandidateLength(2); got != 2 {
		t.Errorf("WordCandidateLength(2) = %d, want 2", got)
	}
	if got := tx.CategoryOfRange(2, 4); got != chars.Katakana {
		t.Errorf("CategoryOfRange(2, 4) = %v, want KATAKANA", got)
	}
}

// FuzzRewrite は任意の置換を適用しても、(空でない) 書き換え後テキスト全体が元テキスト全体に対応し、
// 対応表が単調であることを確かめる。
func FuzzRewrite(f *testing.F) {
	f.Add("ＡＢＣかな漢字", 3, 6, "x")
	f.Add("すごーーーい", 6, 15, "ー")
	f.Add("abc", 0, 3, "")
	f.Fuzz(func(t *testing.T, s string, begin, end int, with string) {
		b := NewBuilder(s)
		if begin < 0 || end > len(s) || begin > end {
			return
		}
		if err := b.Rewrite(func(_ string, e *Editor) error {
			e.Replace(begin, end, with)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		tx := b.Build(simpleCategories)
		if tx.Modified() == "" {
			return
		}
		if ob, oe := tx.OriginalRange(0, len(tx.Modified())); ob != 0 || oe != len(s) {
			t.Fatalf("whole text maps to [%d, %d), want [0, %d)", ob, oe, len(s))
		}
		for i := 1; i < len(tx.m2o); i++ {
			if tx.m2o[i] < tx.m2o[i-1] {
				t.Fatalf("mapping is not monotonic at %d: %v", i, tx.m2o)
			}
		}
	})
}
