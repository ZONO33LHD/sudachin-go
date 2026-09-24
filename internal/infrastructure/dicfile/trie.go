package dicfile

import (
	"encoding/binary"
	"iter"
)

// ダブル配列 (darts-clone 形式) の 1 ユニットのビット配置。
func unitHasLeaf(u uint32) bool  { return (u>>8)&1 == 1 }
func unitValue(u uint32) uint32  { return u & (1<<31 - 1) }
func unitLabel(u uint32) uint32  { return u & (1<<31 | 0xff) }
func unitOffset(u uint32) uint32 { return (u >> 10) << ((u & (1 << 9)) >> 6) }

func (d *Dictionary) unit(i uint32) (uint32, bool) {
	if int(i) >= len(d.trie)/4 {
		return 0, false
	}
	return binary.LittleEndian.Uint32(d.trie[4*i:]), true
}

// trieCommonPrefix は b[offset:] の接頭辞のうちトライに登録されたものについて、(終端バイト位置, 値) を短い順に列挙する。
func (d *Dictionary) trieCommonPrefix(b []byte, offset int) iter.Seq2[int, uint32] {
	return func(yield func(int, uint32) bool) {
		root, ok := d.unit(0)
		if !ok {
			return
		}
		node := unitOffset(root)
		for i := offset; i < len(b); i++ {
			k := uint32(b[i])
			node ^= k
			u, ok := d.unit(node)
			if !ok || unitLabel(u) != k {
				return
			}
			node ^= unitOffset(u)
			if unitHasLeaf(u) {
				leaf, ok := d.unit(node)
				if !ok {
					return
				}
				if !yield(i+1, unitValue(leaf)) {
					return
				}
			}
		}
	}
}
