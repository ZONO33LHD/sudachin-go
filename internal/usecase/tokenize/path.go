package tokenize

import (
	"math"
	"strings"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/lattice"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

// CreatedWords はある位置から始まる語として、どの長さ (文字数) のものが作られたかを表すビット集合。
// 64 文字以上の語はすべて最上位ビットにまとめる。
type CreatedWords uint64

// With は長さ length の語を加えた集合を返す。
func (c CreatedWords) With(length int) CreatedWords {
	return c | 1<<min(max(length-1, 0), 63)
}

// Empty は語が 1 つも作られていないかを返す。
func (c CreatedWords) Empty() bool { return c == 0 }

// PathNode は最小コスト経路上の 1 ノードと、その語の情報。BeginByte/EndByte は正規化後テキストのバイト位置。
type PathNode struct {
	lattice.Node
	BeginByte, EndByte int
	Info               word.Info
}

// joinedParam は連結・分割で作ったノードに与えるパラメータ。コスト計算には使われない。
var joinedParam = word.Param{LeftID: math.MaxUint16, RightID: math.MaxUint16, Cost: math.MaxInt16}

// JoinNodes は nodes を 1 つのノードに連結する。
// 各形は構成ノードの連結になり、normalized が空でなければ正規化形はそれで置き換える。
func JoinNodes(nodes []PathNode, normalized string) PathNode {
	var surface, reading, dicForm, norm strings.Builder
	var headLen uint16
	for _, n := range nodes {
		surface.WriteString(n.Info.Surface)
		reading.WriteString(n.Info.ReadingForm)
		dicForm.WriteString(n.Info.DictionaryForm)
		norm.WriteString(n.Info.NormalizedForm)
		headLen += n.Info.HeadWordLength
	}
	if normalized == "" {
		normalized = norm.String()
	}
	first, last := nodes[0], nodes[len(nodes)-1]
	return PathNode{
		Begin: first.Begin, End: last.End, Param: joinedParam, WordID: word.InvalidID,
		BeginByte: first.BeginByte,
		EndByte:   last.EndByte,
		Info: word.Info{
			Surface:              surface.String(),
			HeadWordLength:       headLen,
			POSID:                first.Info.POSID,
			NormalizedForm:       normalized,
			DictionaryFormWordID: -1,
			DictionaryForm:       dicForm.String(),
			ReadingForm:          reading.String(),
		},
	}
}

// JoinOOVNodes は nodes を品詞 posID の 1 語に連結する。各形は正規化後テキストの該当部分になる。
// 構成ノードがすべて辞書語なら語 ID を引き継ぎ、1 つでも未知語を含めば未知語として扱う。
func JoinOOVNodes(t *text.Text, nodes []PathNode, posID uint16) PathNode {
	first, last := nodes[0], nodes[len(nodes)-1]
	id := first.WordID
	for _, n := range nodes[1:] {
		id = max(id, n.WordID)
	}
	if id.IsOOV() {
		id = word.OOVID(posID)
	}
	return PathNode{
		Begin: first.Begin, End: last.End, Param: joinedParam, WordID: id,
		BeginByte: first.BeginByte,
		EndByte:   last.EndByte,
		Info:      word.Synthesized(t.ModifiedSlice(first.BeginByte, last.EndByte), posID),
	}
}
