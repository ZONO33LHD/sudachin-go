package pathrewrite

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/inputtext"
	"github.com/ZONO33LHD/sudachin-go/internal/adapter/plugin/oov"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/morpheme"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/dicfile"
	"github.com/ZONO33LHD/sudachin-go/internal/infrastructure/resource"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// 以下の ref* は Sudachi (Rust 版) のアルゴリズムを、経路をその場で書き換える形のまま写したもの。
// 本体は O(n) で出力を組み立てる形に書き直しているため、両者の結果が一致することを確かめる。

type refJoinNumeric struct{ j *JoinNumeric }

func (r refJoinNumeric) Rewrite(t *text.Text, path []tokenize.PathNode) ([]tokenize.PathNode, error) {
	path = slices.Clone(path)
	splice := func(begin, end int, p *numberParser) {
		if n, ok := r.j.join(path[begin:end], p); ok {
			path = slices.Concat(path[:begin], []tokenize.PathNode{n}, path[end:])
		}
	}
	begin := -1
	commaAsDigit, periodAsDigit := true, true
	p := newNumberParser()
	for i := 0; i < len(path); i++ {
		n := path[i]
		s := n.Info.NormalizedForm
		if t.CategoryOfRange(n.Begin, n.End).Has(chars.Numeric|chars.KanjiNumeric) || (commaAsDigit && s == ",") || (periodAsDigit && s == ".") {
			if begin < 0 {
				p.reset()
				begin = i
			}
			for _, c := range s {
				if !p.append(c) {
					switch p.err {
					case errComma:
						commaAsDigit, i = false, begin-1
					case errPoint:
						periodAsDigit, i = false, begin-1
					}
					begin = -1
					break
				}
			}
			continue
		}
		if begin >= 0 {
			if p.done() {
				splice(begin, i, p)
				i = begin + 1
			} else if prev := path[i-1].Info.NormalizedForm; (p.err == errComma && prev == ",") || (p.err == errPoint && prev == ".") {
				splice(begin, i-1, p)
				i = begin + 2
			}
		}
		begin = -1
		if !commaAsDigit && s != "," {
			commaAsDigit = true
		}
		if !periodAsDigit && s != "." {
			periodAsDigit = true
		}
	}
	if begin >= 0 {
		if p.done() {
			splice(begin, len(path), p)
		} else if last := path[len(path)-1].Info.NormalizedForm; (p.err == errComma && last == ",") || (p.err == errPoint && last == ".") {
			splice(begin, len(path)-1, p)
		}
	}
	return path, nil
}

type refJoinKatakana struct{ k *JoinKatakanaOOV }

func (r refJoinKatakana) Rewrite(t *text.Text, path []tokenize.PathNode) ([]tokenize.PathNode, error) {
	isKatakana := func(n tokenize.PathNode) bool { return t.CategoryOfRange(n.Begin, n.End).Has(chars.Katakana) }
	path = slices.Clone(path)
	for i := 0; i < len(path); i++ {
		n := path[i]
		if !(n.WordID.IsOOV() || n.End-n.Begin < r.k.minLength) || !isKatakana(n) {
			continue
		}
		begin := i
		for begin > 0 && isKatakana(path[begin-1]) {
			begin--
		}
		end := i + 1
		for end < len(path) && isKatakana(path[end]) {
			end++
		}
		for begin < end && t.Category(path[begin].Begin).Has(chars.NoOOVBOW) {
			begin++
		}
		if end-begin > 1 {
			joined := tokenize.JoinOOVNodes(t, path[begin:end], r.k.posID)
			path = slices.Concat(path[:begin], []tokenize.PathNode{joined}, path[end:])
			i = begin + 1
		}
	}
	return path, nil
}

func newTokenizer(t *testing.T, rewriters ...tokenize.PathRewriter) *tokenize.Tokenizer {
	t.Helper()
	dict, err := dicfile.Open("../../../../testdata/system.dic.test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dict.Close() })
	table, err := resource.DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	def, err := resource.DefaultRewriteDef()
	if err != nil {
		t.Fatal(err)
	}
	noun := oov.Candidate{Param: word.Param{LeftID: 8, RightID: 8, Cost: 6000}, POSID: 4}
	numeral := oov.Candidate{Param: word.Param{LeftID: 9, RightID: 9, Cost: 6000}, POSID: 7}
	cands := map[chars.Category][]oov.Candidate{
		chars.Default: {noun}, chars.Katakana: {noun}, chars.Symbol: {noun}, chars.Numeric: {numeral}, chars.KanjiNumeric: {numeral},
	}
	tk, err := tokenize.New(tokenize.Config{
		Lexicon: dict, Grammar: dict, Categorizer: table,
		InputText:    []tokenize.InputTextPlugin{inputtext.NewNormalizer(def.IgnoreNormalize, def.Replace)},
		OOV:          []tokenize.OOVProvider{oov.NewMeCab(table.Info, cands), oov.NewSimple(noun)},
		PathRewriter: rewriters,
	})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestStreamingRewritersMatchReference(t *testing.T) {
	t.Parallel()
	numeric := NewJoinNumeric(7, true)
	katakana := NewJoinKatakanaOOV(4, 3)
	got := newTokenizer(t, numeric, katakana)
	want := newTokenizer(t, refJoinNumeric{numeric}, refJoinKatakana{katakana})

	alphabet := []rune("0123456789,.，．一二三四六十百千万億〇アイウカキーァ東京都にた、")
	rng := rand.New(rand.NewPCG(1, 2))
	joined := 0
	for range 3000 {
		rs := make([]rune, 1+rng.N(24))
		for i := range rs {
			rs[i] = alphabet[rng.N(len(alphabet))]
		}
		s := string(rs)
		g, err := got.Tokenize(s, morpheme.ModeC)
		if err != nil {
			t.Fatal(err)
		}
		w, err := want.Tokenize(s, morpheme.ModeC)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g, w) {
			t.Fatalf("Tokenize(%q)\n got: %+v\nwant: %+v", s, g, w)
		}
		for _, m := range g {
			if m.OOV && utf8.RuneCountInString(m.Surface) > 1 {
				joined++
			}
		}
	}
	// 連結が実際に起きる入力で比較できていることを確認する
	if joined < 1000 {
		t.Errorf("only %d joined morphemes were produced; the comparison is too weak", joined)
	}
}
