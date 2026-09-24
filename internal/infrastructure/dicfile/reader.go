package dicfile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

var errTruncated = errors.New("dictionary data is truncated")

// reader はリトルエンディアンのバイナリを先頭から読む。範囲外の読み出しは最初の 1 回でエラーを記録し、以降は無視する。
type reader struct {
	data []byte
	pos  int
	err  error
}

func newReader(data []byte, pos int) *reader { return &reader{data: data, pos: pos} }

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.pos+n > len(r.data) {
		r.err = fmt.Errorf("%w: need %d bytes at offset %d, have %d", errTruncated, n, r.pos, len(r.data))
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u8() uint8 {
	if b := r.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *reader) u16() uint16 {
	if b := r.take(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}

func (r *reader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}

func (r *reader) u64() uint64 {
	if b := r.take(8); b != nil {
		return binary.LittleEndian.Uint64(b)
	}
	return 0
}

// varLen は 1 バイト目の最上位ビットが立っていれば 2 バイトで表す長さを読む。
func (r *reader) varLen() int {
	first := r.u8()
	if first < 0x80 {
		return int(first)
	}
	return int(first&0x7f)<<8 | int(r.u8())
}

// utf16String は長さ (UTF-16 の符号単位数) 付きの UTF-16LE 文字列を読む。
func (r *reader) utf16String() string {
	n := r.varLen()
	b := r.take(2 * n)
	if b == nil || n == 0 {
		return ""
	}
	units := make([]uint16, n)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(units))
}

// u32Array は要素数 (1 バイト) 付きの uint32 配列を読み、要素型 T (word.ID など) のスライスとして返す。
func (r *reader) u32Array[T ~uint32]() []T {
	n := int(r.u8())
	if n == 0 {
		return nil
	}
	b := r.take(4 * n)
	if b == nil {
		return nil
	}
	out := make([]T, n)
	for i := range out {
		out[i] = T(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return out
}
