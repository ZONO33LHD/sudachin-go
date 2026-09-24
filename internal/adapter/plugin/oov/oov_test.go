package oov

import (
	"slices"
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/lattice"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/resource"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

func build(t *testing.T, s string) (*text.Text, *chars.Table) {
	t.Helper()
	table, err := resource.DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	return text.NewBuilder(s).Build(table), table
}

type span struct{ begin, end int }

func spans(nodes []lattice.Node) []span {
	out := make([]span, len(nodes))
	for i, n := range nodes {
		out[i] = span{n.Begin, n.End}
	}
	return out
}

func TestMeCab(t *testing.T) {
	t.Parallel()
	cand := []Candidate{{POSID: 1}}
	tests := []struct {
		name     string
		in       string
		existing tokenize.CreatedWords
		want     []span
	}{
		// KATAKANA: INVOKE=1 GROUP=1 LENGTH=2 → 連続部分全体 + 1〜2 文字 (全体と同じ長さは除く)
		{name: "katakana group and prefixes", in: "アイウ", want: []span{{0, 3}, {0, 1}, {0, 2}}},
		// KANJI: INVOKE=0 GROUP=0 LENGTH=2
		{name: "kanji prefixes", in: "漢字", want: []span{{0, 1}, {0, 2}}},
		{name: "kanji is not invoked when a word exists", in: "漢字", existing: tokenize.CreatedWords(0).With(2), want: nil},
		// ALPHA: INVOKE=1 なので既存の語があっても作る
		{name: "alpha is always invoked", in: "abc", existing: tokenize.CreatedWords(0).With(1), want: []span{{0, 3}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx, table := build(t, tt.in)
			m := NewMeCab(table.Info, map[chars.Category][]Candidate{chars.Katakana: cand, chars.Kanji: cand, chars.Alpha: cand})
			got := spans(m.ProvideOOV(tx, 0, tt.existing, nil))
			if !slices.Equal(got, tt.want) {
				t.Errorf("ProvideOOV(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSimple(t *testing.T) {
	t.Parallel()
	tx, _ := build(t, "abc漢")
	s := NewSimple(Candidate{POSID: 2})
	got := spans(s.ProvideOOV(tx, 0, 0, nil))
	// 英字の途中からは語を始められないので、次に始められる位置 (漢) までを 1 語にする
	if len(got) != 1 || got[0] != (span{0, 3}) {
		t.Errorf("ProvideOOV = %v, want [{0 3}]", got)
	}
	if got := s.ProvideOOV(tx, 0, tokenize.CreatedWords(0).With(1), nil); len(got) != 0 {
		t.Errorf("ProvideOOV with existing words = %v, want none", got)
	}
}
