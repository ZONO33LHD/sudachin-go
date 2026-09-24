// Package chars は文字種 (char.def の CATEGORY) のドメインモデルを定義する。
package chars

import (
	"iter"
	"math/bits"
	"slices"
	"strings"
)

// Category は文字種のビット集合。1 文字が複数の文字種に属しうる。
type Category uint32

const (
	Default      Category = 1 << 0
	Space        Category = 1 << 1
	Kanji        Category = 1 << 2
	Symbol       Category = 1 << 3
	Numeric      Category = 1 << 4
	Alpha        Category = 1 << 5
	Hiragana     Category = 1 << 6
	Katakana     Category = 1 << 7
	KanjiNumeric Category = 1 << 8
	Greek        Category = 1 << 9
	Cyrillic     Category = 1 << 10
	User1        Category = 1 << 11
	User2        Category = 1 << 12
	User3        Category = 1 << 13
	User4        Category = 1 << 14
	NoOOVBOW     Category = 1 << 30
	NoOOVBOW2    Category = 1 << 31

	// All は NOOOVBOW 系を除く全ての文字種。
	All Category = 1<<30 - 1
)

var names = map[string]Category{
	"DEFAULT": Default, "SPACE": Space, "KANJI": Kanji, "SYMBOL": Symbol,
	"NUMERIC": Numeric, "ALPHA": Alpha, "HIRAGANA": Hiragana, "KATAKANA": Katakana,
	"KANJINUMERIC": KanjiNumeric, "GREEK": Greek, "CYRILLIC": Cyrillic,
	"USER1": User1, "USER2": User2, "USER3": User3, "USER4": User4,
	"NOOOVBOW": NoOOVBOW, "NOOOVBOW2": NoOOVBOW2, "ALL": All,
}

// ParseCategory は char.def 上の名前から Category を得る。
func ParseCategory(name string) (Category, bool) {
	c, ok := names[name]
	return c, ok
}

// Has は c が other のいずれかのビットを含むかを返す。
func (c Category) Has(other Category) bool { return c&other != 0 }

// Flags は含まれる単一ビットを下位ビットから順に列挙する。
func (c Category) Flags() iter.Seq[Category] {
	return func(yield func(Category) bool) {
		for rest := uint32(c); rest != 0; rest &= rest - 1 {
			if !yield(Category(1) << bits.TrailingZeros32(rest)) {
				return
			}
		}
	}
}

func (c Category) String() string {
	var parts []string
	for f := range c.Flags() {
		for name, v := range names {
			if v == f {
				parts = append(parts, name)
			}
		}
	}
	slices.Sort(parts)
	return strings.Join(parts, "|")
}

// Info は char.def の文字種定義行 (INVOKE / GROUP / LENGTH)。
type Info struct {
	Category Category
	Invoke   bool
	Group    bool
	Length   int
}

// Categorizer は文字から文字種を引く。
type Categorizer interface {
	Category(r rune) Category
}
