// Package word は辞書エントリ（語）に関する値オブジェクトを定義する。
package word

import (
	"fmt"
	"strings"
)

// ID は辞書番号 (上位 4bit) と語番号 (下位 28bit) を 1 つの uint32 に詰めた語の識別子。
// 辞書番号 0xF は未知語 (OOV) を表し、その場合の語番号は品詞 ID になる。
type ID uint32

const (
	dicShift   = 28
	wordMask   = 0x0fff_ffff
	oovDicID   = 0xf
	maxDicID   = 0xe
	MaxWordNum = wordMask
)

// InvalidID は実在の語に対応しない ID。連結ノードなど辞書に存在しない語に使う。
const InvalidID ID = 0xffff_ffff

// NewID は辞書番号と語番号から ID を作る。範囲外の値はエラーにする。
func NewID(dic uint8, num uint32) (ID, error) {
	if dic > maxDicID {
		return InvalidID, fmt.Errorf("dictionary id %d exceeds %d", dic, maxDicID)
	}
	if num > MaxWordNum {
		return InvalidID, fmt.Errorf("word number %d exceeds %d", num, MaxWordNum)
	}
	return ID(uint32(dic)<<dicShift | num), nil
}

// OOVID は品詞 ID を持つ未知語の ID を作る。
func OOVID(posID uint16) ID {
	return ID(oovDicID<<dicShift | uint32(posID))
}

// Dic は辞書番号を返す。
func (id ID) Dic() uint8 { return uint8(id >> dicShift) }

// Num は辞書内の語番号を返す。
func (id ID) Num() uint32 { return uint32(id) & wordMask }

// IsOOV は未知語 (もしくは InvalidID) かどうかを返す。
func (id ID) IsOOV() bool { return id.Dic() == oovDicID }

// Param は連接コスト計算に使う語のパラメータ。
type Param struct {
	LeftID  uint16
	RightID uint16
	Cost    int16
}

// POSDepth は Sudachi の品詞階層の深さ。
const POSDepth = 6

// POS は 6 階層の品詞。配列なので値としてコピー・比較できる。
type POS [POSDepth]string

// ParsePOS は要素数 6 のスライスから POS を作る。
func ParsePOS(parts []string) (POS, error) {
	var p POS
	if len(parts) != POSDepth {
		return p, fmt.Errorf("part of speech must have %d elements, got %d: %v", POSDepth, len(parts), parts)
	}
	copy(p[:], parts)
	return p, nil
}

// String は Sudachi の出力形式 (カンマ区切り) で品詞を返す。
func (p POS) String() string { return strings.Join(p[:], ",") }

// Info は語の詳細情報。辞書から読み出したもの、または未知語・連結語として合成したものを表す。
type Info struct {
	Surface        string
	HeadWordLength uint16
	POSID          uint16
	NormalizedForm string
	// DictionaryFormWordID は辞書形 (終止形など) の語番号。-1 なら自身が辞書形。
	DictionaryFormWordID int32
	DictionaryForm       string
	ReadingForm          string
	AUnitSplit           []ID
	BUnitSplit           []ID
	WordStructure        []ID
	SynonymGroupIDs      []uint32
}

// Synthesized は辞書にない語 (未知語や連結語) の Info を、表層形を各形に流用して作る。
func Synthesized(surface string, posID uint16) Info {
	return Info{
		Surface:              surface,
		HeadWordLength:       uint16(len(surface)),
		POSID:                posID,
		NormalizedForm:       surface,
		DictionaryFormWordID: -1,
		DictionaryForm:       surface,
		ReadingForm:          surface,
	}
}
