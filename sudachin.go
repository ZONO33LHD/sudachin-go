// Package sudachin は Sudachi 辞書を使う日本語形態素解析器。
//
//	a, err := sudachin.Open("system_core.dic")
//	if err != nil { ... }
//	defer a.Close()
//	ms, err := a.Analyze("東京都に住んでいます", sudachin.ModeC)
//
// このパッケージは依存関係を組み立てる入口 (コンポジションルート) で、
// 解析のロジックは internal/usecase/tokenize、辞書の読み込みは internal/infrastructure にある。
package sudachin

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/inputtext"
	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/oov"
	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/pathrewrite"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/morpheme"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/dicfile"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/resource"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

type (
	// Morpheme は解析結果の形態素。
	Morpheme = morpheme.Morpheme
	// Mode は分割単位。
	Mode = morpheme.Mode
	// POS は 6 階層の品詞。
	POS = word.POS
)

// 分割単位。A が最も短く C が最も長い。
const (
	ModeA = morpheme.ModeA
	ModeB = morpheme.ModeB
	ModeC = morpheme.ModeC
)

// ParseMode は "A" / "B" / "C" を Mode に変換する。
func ParseMode(s string) (Mode, error) { return morpheme.ParseMode(s) }

// ErrClosed は Close 済みの Analyzer を使ったことを表す。
var ErrClosed = errors.New("sudachin: analyzer is closed")

// Analyzer は形態素解析器。複数 goroutine から同時に Analyze してよい。
type Analyzer struct {
	dict      *dicfile.Dictionary
	tokenizer *tokenize.Tokenizer
	closed    atomic.Bool
}

type options struct {
	joinNumeric  bool
	normalizeNum bool
	joinKatakana bool
}

// Option は Analyzer の設定。
type Option func(*options)

// WithJoinNumeric は数詞の連結 (と算用数字への正規化) を切り替える。既定は有効。
func WithJoinNumeric(join, normalize bool) Option {
	return func(o *options) { o.joinNumeric, o.normalizeNum = join, normalize }
}

// WithJoinKatakanaOOV はカタカナ未知語の連結を切り替える。既定は有効。
func WithJoinKatakanaOOV(join bool) Option {
	return func(o *options) { o.joinKatakana = join }
}

// Open は dictPath のシステム辞書を開き、Sudachi 既定の設定 (sudachi.json 相当) で Analyzer を作る。
func Open(dictPath string, opts ...Option) (*Analyzer, error) {
	o := options{joinNumeric: true, normalizeNum: true, joinKatakana: true}
	for _, opt := range opts {
		opt(&o)
	}
	dict, err := dicfile.Open(dictPath)
	if err != nil {
		return nil, err
	}
	tk, err := newTokenizer(dict, o)
	if err != nil {
		return nil, errors.Join(err, dict.Close())
	}
	return &Analyzer{dict: dict, tokenizer: tk}, nil
}

// Close は辞書を解放する。Close 以降の Analyze は ErrClosed を返す。
// 辞書は mmap した領域を直接参照しているため、実行中の Analyze と同時に Close してはならない。
func (a *Analyzer) Close() error {
	if a.closed.Swap(true) {
		return nil
	}
	return a.dict.Close()
}

// Analyze は s を形態素に分割する。
func (a *Analyzer) Analyze(s string, mode Mode) ([]Morpheme, error) {
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return a.tokenizer.Tokenize(s, mode)
}

// posIDLookup は品詞から品詞 ID を引く。
type posIDLookup interface {
	POSID(p word.POS) (uint16, bool)
}

// dictionaryInfo は未知語の設定が辞書と矛盾しないかを確かめるのに使う。
type dictionaryInfo interface {
	posIDLookup
	CheckParam(p word.Param) error
}

func lookupPOS(g posIDLookup, parts ...string) (uint16, error) {
	p, err := word.ParsePOS(parts)
	if err != nil {
		return 0, err
	}
	id, ok := g.POSID(p)
	if !ok {
		return 0, fmt.Errorf("part of speech %s is not in the dictionary", p)
	}
	return id, nil
}

func newTokenizer(dict *dicfile.Dictionary, o options) (*tokenize.Tokenizer, error) {
	table, err := resource.DefaultCharTable()
	if err != nil {
		return nil, err
	}
	rewrite, err := resource.DefaultRewriteDef()
	if err != nil {
		return nil, err
	}
	mecab, err := newMeCabOOV(dict, table)
	if err != nil {
		return nil, err
	}
	symbolPOS, err := lookupPOS(dict, "補助記号", "一般", "*", "*", "*", "*")
	if err != nil {
		return nil, err
	}
	simpleParam := word.Param{LeftID: 5968, RightID: 5968, Cost: 3857}
	if err := dict.CheckParam(simpleParam); err != nil {
		return nil, fmt.Errorf("simple OOV: %w", err)
	}

	cfg := tokenize.Config{
		Lexicon:     dict,
		Grammar:     dict,
		Categorizer: table,
		InputText: []tokenize.InputTextPlugin{
			inputtext.NewNormalizer(rewrite.IgnoreNormalize, rewrite.Replace),
			inputtext.NewProlongedSoundMark([]rune{'ー', '-', '⁓', '〜', '〰'}, "ー"),
			inputtext.NewIgnoreYomigana([]rune{'(', '（'}, []rune{')', '）'}, 4,
				func(r rune) bool { return table.Category(r).Has(chars.Kanji) },
				func(r rune) bool { return table.Category(r).Has(chars.Hiragana | chars.Katakana) },
			),
		},
		OOV: []tokenize.OOVProvider{
			mecab,
			oov.NewSimple(oov.Candidate{Param: simpleParam, POSID: symbolPOS}),
		},
	}
	if o.joinNumeric {
		id, err := lookupPOS(dict, "名詞", "数詞", "*", "*", "*", "*")
		if err != nil {
			return nil, err
		}
		cfg.PathRewriter = append(cfg.PathRewriter, pathrewrite.NewJoinNumeric(id, o.normalizeNum))
	}
	if o.joinKatakana {
		id, err := lookupPOS(dict, "名詞", "普通名詞", "一般", "*", "*", "*")
		if err != nil {
			return nil, err
		}
		cfg.PathRewriter = append(cfg.PathRewriter, pathrewrite.NewJoinKatakanaOOV(id, 3))
	}
	return tokenize.New(cfg)
}

func newMeCabOOV(dict dictionaryInfo, table *chars.Table) (*oov.MeCab, error) {
	defs, err := resource.DefaultUnkDef()
	if err != nil {
		return nil, err
	}
	cands := map[chars.Category][]oov.Candidate{}
	for _, d := range defs {
		id, ok := dict.POSID(d.POS)
		if !ok {
			return nil, fmt.Errorf("unk.def: part of speech %s is not in the dictionary", d.POS)
		}
		if err := dict.CheckParam(d.Param); err != nil {
			return nil, fmt.Errorf("unk.def: %s: %w", d.POS, err)
		}
		cands[d.Category] = append(cands[d.Category], oov.Candidate{Param: d.Param, POSID: id})
	}
	return oov.NewMeCab(table.Info, cands), nil
}
