package pathrewrite

import (
	"strconv"
	"strings"
)

// decimal は漢数字・算用数字の解析途中の値を、有効数字の文字列と 10 の冪 (scale)、小数点位置 (point) で表す。
type decimal struct {
	digits  string
	scale   int
	point   int // 小数点が digits の何文字目の前にあるか。なければ -1
	allZero bool
}

func newDecimal() decimal { return decimal{point: -1, allZero: true} }

func (d *decimal) isZero() bool { return d.digits == "" }

func (d *decimal) appendDigit(n int) {
	if n != 0 {
		d.allZero = false
	}
	d.digits += strconv.Itoa(n)
}

func (d *decimal) shiftScale(n int) {
	if d.isZero() {
		d.digits = "1"
	}
	d.scale += n
}

func (d *decimal) normalizeScale() {
	if d.point < 0 {
		return
	}
	fraction := len(d.digits) - d.point
	if fraction > d.scale {
		d.point += d.scale
		d.scale = 0
		return
	}
	d.scale -= fraction
	d.point = -1
}

func (d *decimal) intLength() int {
	d.normalizeScale()
	if d.point >= 0 {
		return d.point
	}
	return len(d.digits) + d.scale
}

// add は o を加える。桁が重なって単純な連結で表せない場合は false を返す。
func (d *decimal) add(o decimal) bool {
	if o.isZero() {
		return true
	}
	if d.isZero() {
		d.digits += o.digits
		d.scale, d.point = o.scale, o.point
		return true
	}
	d.normalizeScale()
	l := o.intLength()
	if d.scale < l {
		return false
	}
	d.digits += strings.Repeat("0", d.scale-l)
	if o.point >= 0 {
		d.point = len(d.digits) + o.point
	}
	d.digits += o.digits
	d.scale = o.scale
	return true
}

func (d *decimal) setPoint() bool {
	if d.scale != 0 || d.point >= 0 {
		return false
	}
	d.point = len(d.digits)
	return true
}

func (d decimal) String() string {
	if d.isZero() {
		return "0"
	}
	d.normalizeScale()
	if d.scale > 0 {
		return d.digits + strings.Repeat("0", d.scale)
	}
	if d.point < 0 {
		return d.digits
	}
	s := d.digits[:d.point] + "." + d.digits[d.point:]
	if d.point == 0 {
		s = "0" + s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

type parseError int

const (
	errNone parseError = iota
	errPoint
	errComma
)

// numberParser は「一万二千」「1,234.5」などの数表記を 1 文字ずつ受け取り、正規化した数値文字列を作る。
type numberParser struct {
	digitLen        int
	isFirstDigit    bool
	hasComma        bool
	hasHangingPoint bool
	err             parseError
	total, subtotal decimal
	tmp             decimal
}

// 正の値は数字、負の値は単位 (十=-1 は 10^1 を表す)。
var numeralValues = map[rune]int{
	'〇': 0, '一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	'十': -1, '百': -2, '千': -3, '万': -4, '億': -8, '兆': -12,
}

func newNumberParser() *numberParser {
	return &numberParser{isFirstDigit: true, total: newDecimal(), subtotal: newDecimal(), tmp: newDecimal()}
}

func (p *numberParser) reset() { *p = *newNumberParser() }

func (p *numberParser) append(c rune) bool {
	switch c {
	case ',':
		if !p.checkComma() {
			p.err = errComma
			return false
		}
		p.hasComma = true
		p.digitLen = 0
		return true
	case '.':
		if p.hasHangingPoint || p.isFirstDigit {
			p.err = errPoint
			return false
		}
		p.hasHangingPoint = true
		if p.hasComma && !p.checkComma() {
			p.err = errComma
			return false
		}
		if !p.tmp.setPoint() {
			p.err = errPoint
			return false
		}
		p.hasComma = false
		return true
	}

	n, ok := numeralValues[c]
	switch {
	case !ok:
		return false
	case n >= -3 && n < 0: // 十・百・千
		p.tmp.shiftScale(-n)
		if !p.subtotal.add(p.tmp) {
			return false
		}
		p.tmp = newDecimal()
		p.startUnit()
	case n < -3: // 万・億・兆
		if !p.subtotal.add(p.tmp) || p.subtotal.isZero() {
			return false
		}
		p.subtotal.shiftScale(-n)
		if !p.total.add(p.subtotal) {
			return false
		}
		p.subtotal, p.tmp = newDecimal(), newDecimal()
		p.startUnit()
	default:
		p.tmp.appendDigit(n)
		p.isFirstDigit = false
		p.digitLen++
		p.hasHangingPoint = false
	}
	return true
}

func (p *numberParser) startUnit() {
	p.isFirstDigit = true
	p.digitLen = 0
	p.hasComma = false
}

func (p *numberParser) checkComma() bool {
	switch {
	case p.isFirstDigit:
		return false
	case !p.hasComma:
		return p.digitLen <= 3 && !p.tmp.isZero() && !p.tmp.allZero
	default:
		return p.digitLen == 3
	}
}

// done は入力の終わりを通知し、全体が数として解釈できたかを返す。
func (p *numberParser) done() bool {
	ok := p.subtotal.add(p.tmp) && p.total.add(p.subtotal)
	if p.hasHangingPoint {
		p.err = errPoint
		return false
	}
	if p.hasComma && p.digitLen != 3 {
		p.err = errComma
		return false
	}
	return ok && p.err == errNone && !p.total.isZero()
}

func (p *numberParser) normalized() string { return p.total.String() }
