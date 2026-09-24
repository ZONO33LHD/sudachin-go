package tokenize

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/lattice"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/morpheme"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

// ErrInvalidUTF8 は入力が UTF-8 として不正なことを表す。
var ErrInvalidUTF8 = errors.New("input is not valid UTF-8")

// Config は Tokenizer の依存関係。
type Config struct {
	Lexicon     Lexicon
	Grammar     Grammar
	Categorizer chars.Categorizer
	InputText   []InputTextPlugin
	// OOV は順に適用される。どれも語を作らなかった位置では、最後のプロバイダをもう一度だけ試す。
	OOV          []OOVProvider
	PathRewriter []PathRewriter
}

// Tokenizer は形態素解析器。生成後は不変なので、複数 goroutine から同時に Tokenize してよい。
type Tokenizer struct {
	cfg Config
}

// New は Tokenizer を作る。
func New(cfg Config) (*Tokenizer, error) {
	switch {
	case cfg.Lexicon == nil:
		return nil, errors.New("tokenizer requires a lexicon")
	case cfg.Grammar == nil:
		return nil, errors.New("tokenizer requires a grammar")
	case cfg.Categorizer == nil:
		return nil, errors.New("tokenizer requires a character categorizer")
	case len(cfg.OOV) == 0:
		return nil, errors.New("tokenizer requires at least one OOV provider")
	}
	return &Tokenizer{cfg: cfg}, nil
}

// Tokenize は s を分割単位 mode で形態素に分割する。
func (tk *Tokenizer) Tokenize(s string, mode morpheme.Mode) ([]morpheme.Morpheme, error) {
	if !utf8.ValidString(s) {
		return nil, ErrInvalidUTF8
	}
	if s == "" {
		return nil, nil
	}
	t, err := tk.prepare(s)
	if err != nil {
		return nil, err
	}
	if t.CharLen() == 0 {
		return nil, nil
	}
	nodes, err := tk.bestPath(t)
	if err != nil {
		return nil, err
	}
	path, err := tk.toPath(t, nodes)
	if err != nil {
		return nil, err
	}
	for _, r := range tk.cfg.PathRewriter {
		if path, err = r.Rewrite(t, path); err != nil {
			return nil, fmt.Errorf("rewrite path: %w", err)
		}
	}
	if path, err = tk.split(t, path, mode); err != nil {
		return nil, err
	}
	return tk.toMorphemes(t, path)
}

func (tk *Tokenizer) prepare(s string) (*text.Text, error) {
	b := text.NewBuilder(s)
	for _, p := range tk.cfg.InputText {
		if err := b.Rewrite(p.Rewrite); err != nil {
			return nil, fmt.Errorf("rewrite input text: %w", err)
		}
	}
	return b.Build(tk.cfg.Categorizer), nil
}

func (tk *Tokenizer) bestPath(t *text.Text) ([]lattice.Node, error) {
	lat := lattice.New(t.CharLen())
	input := []byte(t.Modified())
	var oovs []lattice.Node

	for pos := range t.CharLen() {
		if !lat.HasNodeEndingAt(pos) {
			continue
		}
		byteOff := t.ByteOffset(pos)

		var created CreatedWords
		for e := range tk.cfg.Lexicon.CommonPrefix(input, byteOff) {
			if e.End < len(input) && !t.CanBOW(e.End) {
				continue
			}
			end := t.CharIndex(e.End)
			lat.Insert(lattice.Node{Begin: pos, End: end, Param: tk.cfg.Lexicon.Param(e.WordID), WordID: e.WordID}, tk.cfg.Grammar)
			created = created.With(end - pos)
		}

		if !t.Category(pos).Has(chars.NoOOVBOW | chars.NoOOVBOW2) {
			for _, p := range tk.cfg.OOV {
				oovs, created = tk.provideOOV(p, t, pos, created, lat, oovs[:0])
			}
		}
		if created.Empty() {
			oovs, created = tk.provideOOV(tk.cfg.OOV[len(tk.cfg.OOV)-1], t, pos, created, lat, oovs[:0])
		}
		if created.Empty() {
			return nil, fmt.Errorf("no word can start at character %d (%q)", pos, t.Char(pos))
		}
	}

	nodes, err := lat.BestPath(tk.cfg.Grammar)
	if err != nil {
		return nil, fmt.Errorf("find best path: %w", err)
	}
	return nodes, nil
}

func (tk *Tokenizer) provideOOV(p OOVProvider, t *text.Text, pos int, created CreatedWords, lat *lattice.Lattice, buf []lattice.Node) ([]lattice.Node, CreatedWords) {
	buf = p.ProvideOOV(t, pos, created, buf)
	for _, n := range buf {
		lat.Insert(n, tk.cfg.Grammar)
		created = created.With(n.End - n.Begin)
	}
	return buf, created
}

func (tk *Tokenizer) toPath(t *text.Text, nodes []lattice.Node) ([]PathNode, error) {
	path := make([]PathNode, len(nodes))
	for i, n := range nodes {
		begin, end := t.ByteOffset(n.Begin), t.ByteOffset(n.End)
		info, err := tk.info(t, n.WordID, begin, end)
		if err != nil {
			return nil, err
		}
		path[i] = PathNode{Node: n, BeginByte: begin, EndByte: end, Info: info}
	}
	return path, nil
}

func (tk *Tokenizer) info(t *text.Text, id word.ID, begin, end int) (word.Info, error) {
	if id.IsOOV() {
		return word.Synthesized(t.ModifiedSlice(begin, end), uint16(id.Num())), nil
	}
	info, err := tk.cfg.Lexicon.Info(id)
	if err != nil {
		return word.Info{}, fmt.Errorf("read word info %d: %w", id, err)
	}
	return info, nil
}

// split は分割単位 A/B のとき、分割情報を持つ語をより短い語の並びに置き換える。
func (tk *Tokenizer) split(t *text.Text, path []PathNode, mode morpheme.Mode) ([]PathNode, error) {
	if mode == morpheme.ModeC {
		return path, nil
	}
	out := make([]PathNode, 0, len(path))
	for _, n := range path {
		var ids []word.ID
		switch {
		case n.WordID.IsOOV():
		case mode == morpheme.ModeA:
			ids = n.Info.AUnitSplit
		case mode == morpheme.ModeB:
			ids = n.Info.BUnitSplit
		default:
			return nil, fmt.Errorf("unknown split mode %v", mode)
		}
		if len(ids) <= 1 {
			out = append(out, n)
			continue
		}
		begin := n.BeginByte
		for i, id := range ids {
			info, err := tk.cfg.Lexicon.Info(id)
			if err != nil {
				return nil, fmt.Errorf("read split word info %d: %w", id, err)
			}
			end := begin + int(info.HeadWordLength)
			if i == len(ids)-1 || end > n.EndByte {
				end = n.EndByte
			}
			out = append(out, PathNode{
				Begin: t.CharIndex(begin), End: t.CharIndex(end), Param: joinedParam, WordID: id,
				BeginByte: begin,
				EndByte:   end,
				Info:      info,
			})
			begin = end
		}
	}
	return out, nil
}

func (tk *Tokenizer) toMorphemes(t *text.Text, path []PathNode) ([]morpheme.Morpheme, error) {
	ms := make([]morpheme.Morpheme, len(path))
	for i, n := range path {
		pos, ok := tk.cfg.Grammar.POS(n.Info.POSID)
		if !ok {
			return nil, fmt.Errorf("unknown part of speech id %d", n.Info.POSID)
		}
		begin, end := t.OriginalRange(n.BeginByte, n.EndByte)
		dicID := int(n.WordID.Dic())
		if n.WordID.IsOOV() {
			dicID = -1
		}
		ms[i] = morpheme.Morpheme{
			Surface:         t.Original()[begin:end],
			POS:             pos,
			NormalizedForm:  n.Info.NormalizedForm,
			DictionaryForm:  n.Info.DictionaryForm,
			ReadingForm:     n.Info.ReadingForm,
			DictionaryID:    dicID,
			SynonymGroupIDs: n.Info.SynonymGroupIDs,
			OOV:             n.WordID.IsOOV(),
			Begin:           begin,
			End:             end,
		}
	}
	return ms, nil
}
