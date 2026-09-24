package chars

import (
	"fmt"
	"slices"
)

// Range は [Begin, End) のコードポイント範囲と、その範囲に割り当てる文字種。
type Range struct {
	Begin, End rune
	Category   Category
}

// Table は Range の集合から構築した文字種表。
// 同じ文字に複数の Range が重なる場合は文字種の和集合をとり、どの Range にも属さない文字は Default とする。
type Table struct {
	bmp [0x10000]Category
	// bmp 外の文字用。starts は昇順で、cats[i] は [starts[i], starts[i+1]) の文字種。
	starts []rune
	cats   []Category
	infos  map[Category]Info
}

var _ Categorizer = (*Table)(nil)

// NewTable は範囲定義と文字種定義から Table を構築する。
func NewTable(ranges []Range, infos []Info) (*Table, error) {
	t := &Table{infos: make(map[Category]Info, len(infos))}
	for _, info := range infos {
		t.infos[info.Category] = info
	}

	bounds := []rune{0}
	for _, r := range ranges {
		if r.Begin >= r.End {
			return nil, fmt.Errorf("invalid character range [%#x, %#x)", r.Begin, r.End)
		}
		bounds = append(bounds, r.Begin, r.End)
	}
	slices.Sort(bounds)
	bounds = slices.Compact(bounds)

	cats := make([]Category, len(bounds))
	for _, r := range ranges {
		i, _ := slices.BinarySearch(bounds, r.Begin)
		for ; i < len(bounds) && bounds[i] < r.End; i++ {
			cats[i] |= r.Category
		}
	}
	for i, c := range cats {
		if c == 0 {
			cats[i] = Default
		}
	}
	t.starts, t.cats = bounds, cats

	for cp := range rune(len(t.bmp)) {
		t.bmp[cp] = t.lookup(cp)
	}
	return t, nil
}

func (t *Table) lookup(r rune) Category {
	i, found := slices.BinarySearch(t.starts, r)
	if !found {
		i--
	}
	return t.cats[i]
}

// Category は文字 r の文字種を返す。
func (t *Table) Category(r rune) Category {
	if r >= 0 && int(r) < len(t.bmp) {
		return t.bmp[r]
	}
	return t.lookup(r)
}

// Info は単一の文字種の定義 (INVOKE/GROUP/LENGTH) を返す。
func (t *Table) Info(c Category) (Info, bool) {
	info, ok := t.infos[c]
	return info, ok
}
