package inputtext_test

import (
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/inputtext"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/resource"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

func rewrite(t *testing.T, s string, plugins ...tokenize.InputTextPlugin) *text.Text {
	t.Helper()
	table, err := resource.DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	b := text.NewBuilder(s)
	for _, p := range plugins {
		if err := b.Rewrite(p.Rewrite); err != nil {
			t.Fatal(err)
		}
	}
	return b.Build(table)
}

func defaultNormalizer(t *testing.T) *inputtext.Normalizer {
	t.Helper()
	def, err := resource.DefaultRewriteDef()
	if err != nil {
		t.Fatal(err)
	}
	return inputtext.NewNormalizer(def.IgnoreNormalize, def.Replace)
}

func TestNormalizer(t *testing.T) {
	t.Parallel()
	n := defaultNormalizer(t)
	tests := []struct {
		name, in, want string
	}{
		{name: "full-width alphabet is lowered and NFKC-normalized", in: "ＡＢＣ", want: "abc"},
		{name: "half-width katakana with voiced mark is replaced as a pair", in: "ｶﾞｲﾄﾞ", want: "ガイド"},
		{name: "half-width katakana", in: "ｱｲｳ", want: "アイウ"},
		{name: "roman numerals are Other_Uppercase but excluded from NFKC", in: "Ⅲ", want: "ⅲ"},
		{name: "full-width digits and comma", in: "１２，３００", want: "12,300"},
		{name: "characters in the ignore list are not NFKC-normalized", in: "〜ⅳ", want: "〜ⅳ"},
		{name: "already normalized text", in: "東京都に住む", want: "東京都に住む"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := rewrite(t, tt.in, n)
			if got := tx.Modified(); got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if got := tx.OriginalSlice(0, len(tx.Modified())); got != tt.in {
				t.Errorf("whole text maps back to %q, want %q", got, tt.in)
			}
		})
	}
}

func TestProlongedSoundMark(t *testing.T) {
	t.Parallel()
	p := inputtext.NewProlongedSoundMark([]rune{'ー', '-', '〜'}, "ー")
	tests := []struct{ in, want string }{
		{"すごーーーい", "すごーい"},
		{"ラーメン", "ラーメン"},
		{"a--b〜〜", "aーbー"},
	}
	for _, tt := range tests {
		tx := rewrite(t, tt.in, p)
		if got := tx.Modified(); got != tt.want {
			t.Errorf("rewrite(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// 置換後の文字位置から元テキストを正しく切り出せること (sudachi.go では表層形が欠けていた)
	tx := rewrite(t, "すごーーーい", p)
	if got := tx.OriginalSlice(len("すご"), len("すごーい")); got != "ーーーい" {
		t.Errorf("original of 'ーい' = %q, want %q", got, "ーーーい")
	}
}

func TestIgnoreYomigana(t *testing.T) {
	t.Parallel()
	table, err := resource.DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	p := inputtext.NewIgnoreYomigana([]rune{'(', '（'}, []rune{')', '）'}, 4,
		func(r rune) bool { return table.Category(r).Has(chars.Kanji) },
		func(r rune) bool { return table.Category(r).Has(chars.Hiragana | chars.Katakana) },
	)
	tests := []struct {
		name, in, want string
	}{
		{name: "yomigana after kanji is removed", in: "漢字（かんじ）です", want: "漢字です"},
		{name: "ascii brackets", in: "竹(たけ)", want: "竹"},
		{name: "too long to be yomigana", in: "漢字（かんじかんじ）", want: "漢字（かんじかんじ）"},
		{name: "not after kanji", in: "です（かんじ）", want: "です（かんじ）"},
		{name: "not reading characters", in: "漢字（abc）", want: "漢字（abc）"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := rewrite(t, tt.in, p)
			if got := tx.Modified(); got != tt.want {
				t.Errorf("rewrite(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	// 削除した読み仮名は直前の漢字の表層形に含まれる
	tx := rewrite(t, "漢字（かんじ）です", p)
	if got := tx.OriginalSlice(0, len("漢字")); got != "漢字（かんじ）" {
		t.Errorf("original of '漢字' = %q", got)
	}
	if got := tx.OriginalSlice(len("漢字"), len("漢字です")); got != "です" {
		t.Errorf("original of 'です' = %q", got)
	}
}
