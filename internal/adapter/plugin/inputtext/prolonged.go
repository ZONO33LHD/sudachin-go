package inputtext

import (
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// ProlongedSoundMark は長音記号 (ー や 〜 など) が 2 つ以上連続する部分を 1 つの記号にまとめる。
type ProlongedSoundMark struct {
	marks       map[rune]bool
	replacement string
}

var _ tokenize.InputTextPlugin = (*ProlongedSoundMark)(nil)

// NewProlongedSoundMark は長音記号の集合 marks と置換後の記号 replacement から作る。
func NewProlongedSoundMark(marks []rune, replacement string) *ProlongedSoundMark {
	set := make(map[rune]bool, len(marks))
	for _, m := range marks {
		set[m] = true
	}
	return &ProlongedSoundMark{marks: set, replacement: replacement}
}

// Rewrite は長音記号の連続を置き換える置換を e に登録する。
func (p *ProlongedSoundMark) Rewrite(current string, e *text.Editor) error {
	runStart, runLen := 0, 0
	flush := func(end int) {
		if runLen > 1 {
			e.Replace(runStart, end, p.replacement)
		}
		runLen = 0
	}
	for i, r := range current {
		if !p.marks[r] {
			flush(i)
			continue
		}
		if runLen == 0 {
			runStart = i
		}
		runLen++
	}
	flush(len(current))
	return nil
}
