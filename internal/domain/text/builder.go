// Package text は解析対象テキストのドメインモデルを定義する。
//
// 入力テキストは正規化プラグインによって書き換えられるが、形態素の表層形は常に
// 元のテキストから切り出す必要がある。そのため書き換えは必ず Editor 経由で行い、
// Builder が「書き換え後のバイト位置 → 元テキストのバイト位置」の対応表を維持する。
// 書き換え中 (Builder) と解析用 (Text) を別の型にすることで、状態遷移の誤りをコンパイル時に防ぐ。
package text

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Editor は 1 回の書き換えで適用する置換を集める。位置はすべて現在のテキスト上のバイト位置。
type Editor struct {
	edits []edit
}

type edit struct {
	begin, end int
	with       string
}

// Replace は [begin, end) を with に置き換える。with が空なら削除になる。
func (e *Editor) Replace(begin, end int, with string) {
	e.edits = append(e.edits, edit{begin: begin, end: end, with: with})
}

// Builder は書き換え可能な入力テキスト。
type Builder struct {
	original string
	current  string
	// m2o[i] は current の i バイト目に対応する original のバイト位置。末尾に番兵を持つ。
	m2o []int
}

// NewBuilder は original を初期状態とする Builder を作る。
func NewBuilder(original string) *Builder {
	m2o := make([]int, len(original)+1)
	for i := range m2o {
		m2o[i] = i
	}
	return &Builder{original: original, current: original, m2o: m2o}
}

// Current は現在 (書き換え後) のテキストを返す。
func (b *Builder) Current() string { return b.current }

// Rewrite は fn に現在のテキストと Editor を渡し、集めた置換を一括で適用する。
func (b *Builder) Rewrite(fn func(current string, e *Editor) error) error {
	var e Editor
	if err := fn(b.current, &e); err != nil {
		return err
	}
	if len(e.edits) == 0 {
		return nil
	}
	return b.apply(e.edits)
}

func (b *Builder) apply(edits []edit) error {
	edits = slices.SortedStableFunc(slices.Values(edits), func(x, y edit) int { return cmp.Compare(x.begin, y.begin) })

	var sb strings.Builder
	sb.Grow(len(b.current))
	m2o := make([]int, 0, len(b.m2o))
	pos := 0
	for _, ed := range edits {
		if ed.begin < pos || ed.end < ed.begin || ed.end > len(b.current) {
			return fmt.Errorf("invalid or overlapping edit [%d, %d) at position %d (text length %d)", ed.begin, ed.end, pos, len(b.current))
		}
		sb.WriteString(b.current[pos:ed.begin])
		m2o = append(m2o, b.m2o[pos:ed.begin]...)
		if ed.with != "" {
			sb.WriteString(ed.with)
			// 置換後の先頭バイトは置換元の先頭に、残りのバイトは置換元の末尾に対応付ける。
			m2o = append(m2o, b.m2o[ed.begin])
			for range len(ed.with) - 1 {
				m2o = append(m2o, b.m2o[ed.end])
			}
		}
		pos = ed.end
	}
	sb.WriteString(b.current[pos:])
	m2o = append(m2o, b.m2o[pos:]...)

	// 先頭が削除された場合でも、形態素の表層形を連結すると元テキスト全体になるように先頭を 0 に固定する。
	if len(m2o) > 1 {
		m2o[0] = 0
	}
	b.current = sb.String()
	b.m2o = m2o
	return nil
}
