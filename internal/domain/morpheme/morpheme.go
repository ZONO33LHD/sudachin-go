// Package morpheme は解析結果である形態素と分割単位を定義する。
package morpheme

import (
	"fmt"
	"strings"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

// Mode は分割単位。A が最も短く C が最も長い。
type Mode int

const (
	ModeA Mode = iota + 1
	ModeB
	ModeC
)

// ParseMode は "A" / "B" / "C" (大文字小文字は問わない) を Mode に変換する。
func ParseMode(s string) (Mode, error) {
	switch strings.ToUpper(s) {
	case "A":
		return ModeA, nil
	case "B":
		return ModeB, nil
	case "C":
		return ModeC, nil
	default:
		return 0, fmt.Errorf(`split mode must be "A", "B" or "C", got %q`, s)
	}
}

func (m Mode) String() string {
	switch m {
	case ModeA:
		return "A"
	case ModeB:
		return "B"
	case ModeC:
		return "C"
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

// Morpheme は 1 つの形態素。Begin/End は元テキスト上のバイト位置。
type Morpheme struct {
	Surface         string
	POS             word.POS
	NormalizedForm  string
	DictionaryForm  string
	ReadingForm     string
	DictionaryID    int
	SynonymGroupIDs []uint32
	OOV             bool
	Begin, End      int
}
