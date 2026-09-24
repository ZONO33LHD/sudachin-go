package resource

import (
	"strings"
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
)

func TestDefaultResources(t *testing.T) {
	t.Parallel()
	table, err := DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		r    rune
		want chars.Category
	}{
		{'a', chars.Alpha},
		{'漢', chars.Kanji},
		{'一', chars.Kanji | chars.KanjiNumeric},
		{'ア', chars.Katakana},
		{'ァ', chars.Katakana | chars.NoOOVBOW},
		{'。', chars.Symbol},
	}
	for _, tt := range tests {
		if got := table.Category(tt.r); got != tt.want {
			t.Errorf("Category(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
	if info, ok := table.Info(chars.Katakana); !ok || !info.Invoke || !info.Group || info.Length != 2 {
		t.Errorf("Info(KATAKANA) = %+v, %v", info, ok)
	}

	unk, err := DefaultUnkDef()
	if err != nil {
		t.Fatal(err)
	}
	if len(unk) == 0 || unk[0].Category != chars.Default || unk[0].POS.String() != "補助記号,一般,*,*,*,*" {
		t.Errorf("first unk.def entry = %+v", unk[0])
	}

	rw, err := DefaultRewriteDef()
	if err != nil {
		t.Fatal(err)
	}
	if !rw.IgnoreNormalize['Ⅲ'] || rw.Replace["ｶﾞ"] != "ガ" {
		t.Errorf("rewrite.def was not parsed as expected: ignore Ⅲ=%v, ｶﾞ=%q", rw.IgnoreNormalize['Ⅲ'], rw.Replace["ｶﾞ"])
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		parse func(string) error
		in    string
		want  string
	}{
		{"unknown category in char.def", func(s string) error { _, err := ParseCharDef(strings.NewReader(s)); return err }, "0x0041 LATIN", `line 1: unknown character category "LATIN"`},
		{"bad code point", func(s string) error { _, err := ParseCharDef(strings.NewReader(s)); return err }, "0xZZ ALPHA", "invalid code point"},
		{"short unk.def line", func(s string) error { _, err := ParseUnkDef(strings.NewReader(s)); return err }, "ALPHA,1,1,100", "needs 10 columns"},
		{"duplicated replacement", func(s string) error { _, err := ParseRewriteDef(strings.NewReader(s)); return err }, "a b\na c", "defined twice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.parse(tt.in); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}
