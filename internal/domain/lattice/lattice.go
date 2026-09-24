// Package lattice は形態素ラティスと最小コスト経路探索 (Viterbi) を実装する。
package lattice

import (
	"errors"
	"math"
	"slices"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

// ErrNoPath は BOS から EOS へ到達する経路が存在しないことを表す。
var ErrNoPath = errors.New("lattice has no path from BOS to EOS")

// Node はラティス上のノード。Begin/End は正規化後テキストの文字位置。
type Node struct {
	Begin, End int
	word.Param
	WordID word.ID
}

// Coster は左文脈 ID と右文脈 ID の連接コストを返す。
type Coster interface {
	ConnectCost(left, right uint16) int16
}

const unreachable = math.MaxInt32

type entry struct {
	node      Node
	totalCost int32
	// prev は最小コストで接続する直前ノードの ends[node.Begin] 内の添字。BOS に接続する場合は -1。
	prev int
}

// Lattice は文字位置ごとに「そこで終わるノード」を保持する。
type Lattice struct {
	ends [][]entry
}

// New は charLen 文字のテキスト用のラティスを作る。
func New(charLen int) *Lattice {
	l := &Lattice{ends: make([][]entry, charLen+1)}
	l.ends[0] = []entry{{prev: -1}} // BOS
	return l
}

// HasNodeEndingAt は pos で終わるノード (BOS を含む) があるかを返す。
func (l *Lattice) HasNodeEndingAt(pos int) bool { return len(l.ends[pos]) > 0 }

// Insert はノードを追加し、接続可能な直前ノードのうち累積コスト最小のものに接続する。
// コストが同じ場合は先に追加されたノードを優先する。
func (l *Lattice) Insert(n Node, c Coster) {
	cost, prev := l.connect(n.Begin, n.LeftID, int32(n.Cost), c)
	l.ends[n.End] = append(l.ends[n.End], entry{node: n, totalCost: cost, prev: prev})
}

func (l *Lattice) connect(begin int, leftID uint16, nodeCost int32, c Coster) (int32, int) {
	minCost, prev := int32(unreachable), -1
	for i, e := range l.ends[begin] {
		if e.totalCost == unreachable {
			continue
		}
		cost := e.totalCost + int32(c.ConnectCost(e.node.RightID, leftID)) + nodeCost
		if cost < minCost {
			minCost, prev = cost, i
		}
	}
	return minCost, prev
}

// BestPath は EOS に接続したうえで、BOS から EOS までの最小コスト経路のノードを先頭から順に返す。
func (l *Lattice) BestPath(c Coster) ([]Node, error) {
	eos := len(l.ends) - 1
	cost, prev := l.connect(eos, 0, 0, c)
	if cost == unreachable {
		return nil, ErrNoPath
	}
	var path []Node
	pos, idx := eos, prev
	for pos > 0 {
		e := l.ends[pos][idx]
		path = append(path, e.node)
		pos, idx = e.node.Begin, e.prev
	}
	slices.Reverse(path)
	return path, nil
}
