// Package oov は辞書にない語 (未知語) のノードを生成するプラグインを実装する。
package oov

import (
	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/lattice"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// Candidate は文字種に対応する未知語 1 種類分の品詞とコスト。
type Candidate struct {
	Param word.Param
	POSID uint16
}

// MeCab は MeCab 方式 (char.def / unk.def) の未知語処理を行う。
//
// 文字種ごとに、同じ文字種が続く範囲を 1 語にまとめたもの (GROUP) と、
// 先頭から 1〜LENGTH 文字のもの を未知語候補として生成する。
// INVOKE が偽の文字種は、その位置に既に語がある場合は候補を作らない。
type MeCab struct {
	infos      func(chars.Category) (chars.Info, bool)
	candidates map[chars.Category][]Candidate
}

var _ tokenize.OOVProvider = (*MeCab)(nil)

// NewMeCab は文字種定義の参照関数と、文字種ごとの未知語候補から作る。
func NewMeCab(infos func(chars.Category) (chars.Info, bool), candidates map[chars.Category][]Candidate) *MeCab {
	return &MeCab{infos: infos, candidates: candidates}
}

// ProvideOOV は offset から始まる未知語ノードを out に追加して返す。
func (m *MeCab) ProvideOOV(t *text.Text, offset int, existing tokenize.CreatedWords, out []lattice.Node) []lattice.Node {
	runLen := t.CategoryRun(offset)
	for cat := range t.Category(offset).Flags() {
		info, ok := m.infos(cat)
		if !ok || (!info.Invoke && !existing.Empty()) {
			continue
		}
		cands := m.candidates[cat]
		if len(cands) == 0 {
			continue
		}
		limit := runLen
		if info.Group {
			out = appendNodes(out, cands, offset, offset+runLen)
			limit--
		}
		for l := 1; l <= min(info.Length, limit); l++ {
			out = appendNodes(out, cands, offset, offset+l)
		}
	}
	return out
}

func appendNodes(out []lattice.Node, cands []Candidate, begin, end int) []lattice.Node {
	for _, c := range cands {
		out = append(out, lattice.Node{Begin: begin, End: end, Param: c.Param, WordID: word.OOVID(c.POSID)})
	}
	return out
}

// Simple は他に語が作られなかった位置に、次に語を始められる位置までを 1 語とする未知語を作る。
type Simple struct {
	cand Candidate
}

var _ tokenize.OOVProvider = (*Simple)(nil)

// NewSimple は生成する未知語の品詞とコストから作る。
func NewSimple(c Candidate) *Simple { return &Simple{cand: c} }

// ProvideOOV は offset に語が 1 つもないときだけ未知語ノードを 1 つ追加する。
func (s *Simple) ProvideOOV(t *text.Text, offset int, existing tokenize.CreatedWords, out []lattice.Node) []lattice.Node {
	if !existing.Empty() {
		return out
	}
	end := offset + t.WordCandidateLength(offset)
	return appendNodes(out, []Candidate{s.cand}, offset, end)
}
