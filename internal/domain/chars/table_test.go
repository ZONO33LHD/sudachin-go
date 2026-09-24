package chars

import (
	"slices"
	"testing"
)

func TestTableCategory(t *testing.T) {
	t.Parallel()
	table, err := NewTable([]Range{
		{Begin: 'A', End: 'Z' + 1, Category: Alpha},
		{Begin: '0', End: '9' + 1, Category: Numeric},
		{Begin: '一', End: '一' + 1, Category: Kanji},
		{Begin: '一', End: '一' + 1, Category: KanjiNumeric},
		{Begin: 0x1F600, End: 0x1F650, Category: Symbol},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		r    rune
		want Category
	}{
		{'A', Alpha},
		{'Z', Alpha},
		{'[', Default},
		{'5', Numeric},
		{'一', Kanji | KanjiNumeric},
		{'二', Default},
		{0x1F600, Symbol},
		{0x1F650, Default},
		{0x10FFFF, Default},
	}
	for _, tt := range tests {
		if got := table.Category(tt.r); got != tt.want {
			t.Errorf("Category(%U) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestNewTableRejectsEmptyRange(t *testing.T) {
	t.Parallel()
	if _, err := NewTable([]Range{{Begin: 'b', End: 'a', Category: Alpha}}, nil); err == nil {
		t.Error("NewTable should reject a range whose end precedes its begin")
	}
}

func TestCategoryFlags(t *testing.T) {
	t.Parallel()
	got := slices.Collect((Kanji | Alpha | NoOOVBOW).Flags())
	want := []Category{Kanji, Alpha, NoOOVBOW}
	if !slices.Equal(got, want) {
		t.Errorf("Flags() = %v, want %v", got, want)
	}
}

func TestCategoryString(t *testing.T) {
	t.Parallel()
	if got := (Kanji | KanjiNumeric).String(); got != "KANJI|KANJINUMERIC" {
		t.Errorf("String() = %q", got)
	}
	if c, ok := ParseCategory("KATAKANA"); !ok || c != Katakana {
		t.Errorf("ParseCategory(KATAKANA) = %v, %v", c, ok)
	}
}
