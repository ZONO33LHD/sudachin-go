// Package pathrewrite は最小コスト経路を書き換えるプラグインを実装する。
package pathrewrite

import (
	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

// JoinNumeric は連続する数字・漢数字の語を 1 語 (名詞,数詞) に連結し、正規化形を算用数字にする。
type JoinNumeric struct {
	numericPOSID uint16
	normalize    bool
}

var _ tokenize.PathRewriter = (*JoinNumeric)(nil)

// NewJoinNumeric は数詞の品詞 ID と、正規化形を算用数字に直すかどうかから作る。
func NewJoinNumeric(numericPOSID uint16, normalize bool) *JoinNumeric {
	return &JoinNumeric{numericPOSID: numericPOSID, normalize: normalize}
}

// Rewrite は数詞の連結を行った新しい経路を返す。
//
// 「,」「.」は桁区切り・小数点として数の一部とみなすが、数として解釈できなかった場合は
// それを数の一部とみなさずに同じ位置から解析し直す。連結しなかった場合の走査位置の戻し方も含め、
// Sudachi (Rust 版) の JoinNumericPlugin と同じ結果になるようにしている。
func (j *JoinNumeric) Rewrite(t *text.Text, in []tokenize.PathNode) ([]tokenize.PathNode, error) {
	out := make([]tokenize.PathNode, 0, len(in))
	next := 0 // in[next:] はまだ out に出力していない
	join := func(begin, end int, p *numberParser) bool {
		n, ok := j.join(in[begin:end], p)
		if ok {
			out = append(append(out, in[next:begin]...), n)
			next = end
		}
		return ok
	}
	isSeparator := func(i int, p *numberParser) bool {
		s := in[i].Info.NormalizedForm
		return (p.err == errComma && s == ",") || (p.err == errPoint && s == ".")
	}

	begin := -1
	commaAsDigit, periodAsDigit := true, true
	p := newNumberParser()
	for i := 0; i < len(in); i++ {
		n := in[i]
		s := n.Info.NormalizedForm
		if t.CategoryOfRange(n.Begin, n.End).Has(chars.Numeric|chars.KanjiNumeric) || (commaAsDigit && s == ",") || (periodAsDigit && s == ".") {
			if begin < 0 {
				p.reset()
				begin = i
			}
			for _, c := range s {
				if p.append(c) {
					continue
				}
				switch p.err {
				case errComma:
					commaAsDigit = false
					i = begin - 1
				case errPoint:
					periodAsDigit = false
					i = begin - 1
				}
				begin = -1
				break
			}
			continue
		}

		if begin >= 0 {
			if p.done() {
				if !join(begin, i, p) {
					i = begin + 1
				}
			} else if isSeparator(i-1, p) && !join(begin, i-1, p) {
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
			join(begin, len(in), p)
		} else if isSeparator(len(in)-1, p) {
			join(begin, len(in)-1, p)
		}
	}
	return append(out, in[next:]...), nil
}

// join は nodes を 1 つの数詞に連結する。連結する必要がなければ false を返す。
func (j *JoinNumeric) join(nodes []tokenize.PathNode, p *numberParser) (tokenize.PathNode, bool) {
	if len(nodes) == 0 || nodes[0].Info.POSID != j.numericPOSID {
		return tokenize.PathNode{}, false
	}
	if !j.normalize {
		return tokenize.JoinNodes(nodes, ""), len(nodes) > 1
	}
	normalized := p.normalized()
	if len(nodes) > 1 || normalized != nodes[0].Info.NormalizedForm {
		return tokenize.JoinNodes(nodes, normalized), true
	}
	return tokenize.PathNode{}, false
}

// JoinKatakanaOOV は未知語または短い語を含むカタカナの並びを 1 語に連結する。
type JoinKatakanaOOV struct {
	posID     uint16
	minLength int
}

var _ tokenize.PathRewriter = (*JoinKatakanaOOV)(nil)

// NewJoinKatakanaOOV は連結後の品詞 ID と、未知語でなくても連結対象とする語の最小文字数から作る。
func NewJoinKatakanaOOV(posID uint16, minLength int) *JoinKatakanaOOV {
	return &JoinKatakanaOOV{posID: posID, minLength: minLength}
}

// Rewrite はカタカナ語の連結を行った新しい経路を返す。
func (k *JoinKatakanaOOV) Rewrite(t *text.Text, in []tokenize.PathNode) ([]tokenize.PathNode, error) {
	isKatakana := func(n tokenize.PathNode) bool { return t.CategoryOfRange(n.Begin, n.End).Has(chars.Katakana) }
	out := make([]tokenize.PathNode, 0, len(in))
	next := 0 // in[next:] はまだ out に出力していない
	for i := 0; i < len(in); i++ {
		n := in[i]
		trigger := n.WordID.IsOOV() || n.End-n.Begin < k.minLength
		if !trigger || !isKatakana(n) {
			continue
		}
		begin := i
		for begin > next && isKatakana(in[begin-1]) {
			begin--
		}
		end := i + 1
		for end < len(in) && isKatakana(in[end]) {
			end++
		}
		for begin < end && t.Category(in[begin].Begin).Has(chars.NoOOVBOW) {
			begin++
		}
		if end-begin > 1 {
			out = append(append(out, in[next:begin]...), tokenize.JoinOOVNodes(t, in[begin:end], k.posID))
			next = end
			// 連結直後のノード in[end] はカタカナではないので調べずに飛ばす。
			i = end
		}
	}
	return append(out, in[next:]...), nil
}
