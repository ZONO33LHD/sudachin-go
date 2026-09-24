package tokenize_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

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

// テスト用辞書 (sudachi.rs 由来の 39 語) の品詞 ID。
const (
	posCommonNoun = 4 // 名詞,普通名詞,一般,*,*,*
	posNumeral    = 7 // 名詞,数詞,*,*,*,*
)

func newTestTokenizer(t testing.TB) *tokenize.Tokenizer {
	t.Helper()
	dict, err := dicfile.Open("../../../testdata/system.dic.test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dict.Close() })
	table, err := resource.DefaultCharTable()
	if err != nil {
		t.Fatal(err)
	}
	rewrite, err := resource.DefaultRewriteDef()
	if err != nil {
		t.Fatal(err)
	}
	unknown := oov.Candidate{Param: word.Param{LeftID: 8, RightID: 8, Cost: 6000}, POSID: posCommonNoun}
	cands := map[chars.Category][]oov.Candidate{}
	for _, c := range []chars.Category{chars.Default, chars.Alpha, chars.Katakana, chars.Hiragana, chars.Kanji, chars.Symbol} {
		cands[c] = []oov.Candidate{unknown}
	}
	cands[chars.Numeric] = []oov.Candidate{{Param: word.Param{LeftID: 9, RightID: 9, Cost: 6000}, POSID: posNumeral}}

	tk, err := tokenize.New(tokenize.Config{
		Lexicon:     dict,
		Grammar:     dict,
		Categorizer: table,
		InputText: []tokenize.InputTextPlugin{
			inputtext.NewNormalizer(rewrite.IgnoreNormalize, rewrite.Replace),
			inputtext.NewProlongedSoundMark([]rune{'ー', '〜'}, "ー"),
		},
		OOV:          []tokenize.OOVProvider{oov.NewMeCab(table.Info, cands), oov.NewSimple(unknown)},
		PathRewriter: []tokenize.PathRewriter{pathrewrite.NewJoinNumeric(posNumeral, true), pathrewrite.NewJoinKatakanaOOV(posCommonNoun, 3)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func surfaces(ms []morpheme.Morpheme) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Surface
	}
	return out
}

func TestTokenize(t *testing.T) {
	t.Parallel()
	tk := newTestTokenizer(t)
	tests := []struct {
		name string
		in   string
		mode morpheme.Mode
		want []string
	}{
		{name: "dictionary words", in: "東京都に行った", mode: morpheme.ModeC, want: []string{"東京都", "に", "行っ", "た"}},
		{name: "mode A splits a compound", in: "東京都に行った", mode: morpheme.ModeA, want: []string{"東京", "都", "に", "行っ", "た"}},
		{name: "mode B keeps a compound without B split", in: "東京都", mode: morpheme.ModeB, want: []string{"東京都"}},
		{name: "surface comes from the original text", in: "東京都にイッた", mode: morpheme.ModeC, want: []string{"東京都", "に", "イッ", "た"}},
		{name: "numerals are joined", in: "東京都に１２３", mode: morpheme.ModeC, want: []string{"東京都", "に", "１２３"}},
		{name: "unknown alphabet is grouped", in: "ABCに行く", mode: morpheme.ModeC, want: []string{"ABC", "に", "行く"}},
		{name: "prolonged sound marks keep the original surface", in: "京都ーーー", mode: morpheme.ModeC, want: []string{"京都", "ーーー"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms, err := tk.Tokenize(tt.in, tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if got := surfaces(ms); strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("Tokenize(%q, %v) = %v, want %v", tt.in, tt.mode, got, tt.want)
			}
		})
	}
}

func TestTokenizeFields(t *testing.T) {
	t.Parallel()
	tk := newTestTokenizer(t)

	ms, err := tk.Tokenize("東京都に１２３", morpheme.ModeC)
	if err != nil {
		t.Fatal(err)
	}
	num := ms[len(ms)-1]
	if num.NormalizedForm != "123" || !num.OOV || num.DictionaryID != -1 {
		t.Errorf("joined numeral = %+v, want normalized 123 and OOV", num)
	}
	if got := num.POS.String(); got != "名詞,数詞,*,*,*,*" {
		t.Errorf("joined numeral POS = %s", got)
	}
	if ms[0].Begin != 0 || ms[0].End != len("東京都") || ms[0].DictionaryID != 0 {
		t.Errorf("first morpheme = %+v", ms[0])
	}
}

func TestTokenizeInvalidInput(t *testing.T) {
	t.Parallel()
	tk := newTestTokenizer(t)
	if _, err := tk.Tokenize("\xff\xfe", morpheme.ModeC); !errors.Is(err, tokenize.ErrInvalidUTF8) {
		t.Errorf("Tokenize(invalid UTF-8) error = %v, want ErrInvalidUTF8", err)
	}
	ms, err := tk.Tokenize("", morpheme.ModeC)
	if err != nil || len(ms) != 0 {
		t.Errorf("Tokenize(\"\") = %v, %v, want no morphemes", ms, err)
	}
}

// Tokenizer は不変なので、同じインスタンスを複数 goroutine から使える (go test -race で確認する)。
func TestTokenizeConcurrently(t *testing.T) {
	t.Parallel()
	tk := newTestTokenizer(t)
	want, err := tk.Tokenize("東京都に行った", morpheme.ModeA)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 50 {
				got, err := tk.Tokenize("東京都に行った", morpheme.ModeA)
				if err != nil {
					t.Error(err)
					return
				}
				if len(got) != len(want) {
					t.Errorf("concurrent Tokenize returned %d morphemes, want %d", len(got), len(want))
					return
				}
			}
		})
	}
	wg.Wait()
}

// FuzzTokenize はどんな入力でも panic せず、表層形を連結すると入力に戻ることを確かめる。
func FuzzTokenize(f *testing.F) {
	for _, s := range []string{"東京都に行った", "ＡＢＣｶﾞｲﾄﾞ１，２３４．５", "すごーーーい", "a\u0301\u200d\U0001F44D\U0001F3FB", "漢字（かんじ）"} {
		f.Add(s)
	}
	tk := newTestTokenizer(f)
	f.Fuzz(func(t *testing.T, s string) {
		for _, mode := range []morpheme.Mode{morpheme.ModeA, morpheme.ModeC} {
			ms, err := tk.Tokenize(s, mode)
			if errors.Is(err, tokenize.ErrInvalidUTF8) {
				return
			}
			if err != nil {
				t.Fatalf("Tokenize(%q) failed: %v", s, err)
			}
			if got := strings.Join(surfaces(ms), ""); got != s {
				t.Fatalf("surfaces of %q join to %q", s, got)
			}
		}
	})
}
