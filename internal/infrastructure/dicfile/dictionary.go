// Package dicfile は Sudachi 形式のバイナリ辞書 (system.dic) を読み込み、tokenize のポートを実装する。
package dicfile

import (
	"encoding/binary"
	"fmt"
	"iter"
	"time"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
	"github.com/ZONO33LHD/sudachin-go/internal/usecase/tokenize"
)

const (
	systemDictV1 uint64 = 0x7366d3f18bd111e7
	systemDictV2 uint64 = 0xce9f011a92394434

	descriptionSize = 256
)

// Header は辞書ファイルのヘッダ。
type Header struct {
	Version     uint64
	CreatedAt   time.Time
	Description string
}

// Dictionary はシステム辞書。辞書データ (mmap されたバイト列) を参照するだけで、生成後は変更されない。
type Dictionary struct {
	Header Header

	data    []byte
	release func() error

	posList []word.POS
	posIDs  map[word.POS]uint16
	matrix  []byte
	numLeft int

	trie         []byte // uint32 の配列
	wordIDTable  []byte
	params       []byte // (left, right, cost) の int16 x3 の配列
	infoOffsets  []byte // uint32 の配列
	numWords     int
	hasSynonymID bool
}

var (
	_ tokenize.Lexicon = (*Dictionary)(nil)
	_ tokenize.Grammar = (*Dictionary)(nil)
)

// Open は path のシステム辞書を開く。使い終わったら Close すること。
func Open(path string) (*Dictionary, error) {
	data, release, err := mapFile(path)
	if err != nil {
		return nil, fmt.Errorf("open dictionary %s: %w", path, err)
	}
	d, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse dictionary %s: %w", path, err)
	}
	d.release = release
	return d, nil
}

// Close は辞書データを解放する。
func (d *Dictionary) Close() error {
	if d.release == nil {
		return nil
	}
	release := d.release
	d.release = nil
	return release()
}

// Parse はメモリ上の辞書データを解析する。data は Dictionary が使われている間は変更してはならない。
func Parse(data []byte) (*Dictionary, error) {
	d := &Dictionary{data: data}
	r := newReader(data, 0)
	d.Header.Version = r.u64()
	created := r.u64()
	desc := r.take(descriptionSize)
	if r.err != nil {
		return nil, fmt.Errorf("read header: %w", r.err)
	}
	switch d.Header.Version {
	case systemDictV1:
	case systemDictV2:
		d.hasSynonymID = true
	default:
		return nil, fmt.Errorf("unsupported dictionary version %#016x (only system dictionaries are supported)", d.Header.Version)
	}
	d.Header.CreatedAt = time.Unix(int64(created), 0)
	d.Header.Description = cString(desc)

	if err := d.parseGrammar(r); err != nil {
		return nil, fmt.Errorf("read grammar: %w", err)
	}
	if err := d.parseLexicon(r); err != nil {
		return nil, fmt.Errorf("read lexicon: %w", err)
	}
	return d, nil
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func (d *Dictionary) parseGrammar(r *reader) error {
	n := int(r.u16())
	d.posList = make([]word.POS, n)
	d.posIDs = make(map[word.POS]uint16, n)
	for i := range n {
		var p word.POS
		for j := range p {
			p[j] = r.utf16String()
		}
		d.posList[i] = p
		if _, dup := d.posIDs[p]; !dup {
			d.posIDs[p] = uint16(i)
		}
	}
	d.numLeft = int(r.u16())
	numRight := int(r.u16())
	d.matrix = r.take(2 * d.numLeft * numRight)
	return r.err
}

func (d *Dictionary) parseLexicon(r *reader) error {
	trieSize := int(r.u32())
	d.trie = r.take(4 * trieSize)
	tableSize := int(r.u32())
	d.wordIDTable = r.take(tableSize)
	d.numWords = int(r.u32())
	d.params = r.take(6 * d.numWords)
	d.infoOffsets = r.take(4 * d.numWords)
	if r.err != nil {
		return r.err
	}
	if trieSize == 0 {
		return fmt.Errorf("trie is empty")
	}
	return nil
}

// ConnectCost は左ノードの右文脈 ID と右ノードの左文脈 ID の連接コストを返す。
func (d *Dictionary) ConnectCost(left, right uint16) int16 {
	i := 2 * (int(right)*d.numLeft + int(left))
	if i+2 > len(d.matrix) {
		return 0
	}
	return int16(binary.LittleEndian.Uint16(d.matrix[i:]))
}

// POS は品詞 ID に対応する品詞を返す。
func (d *Dictionary) POS(id uint16) (word.POS, bool) {
	if int(id) >= len(d.posList) {
		return word.POS{}, false
	}
	return d.posList[id], true
}

// POSID は品詞に対応する品詞 ID を返す。
func (d *Dictionary) POSID(p word.POS) (uint16, bool) {
	id, ok := d.posIDs[p]
	return id, ok
}

// Param は語の連接パラメータを返す。
func (d *Dictionary) Param(id word.ID) word.Param {
	i := 6 * int(id.Num())
	if i+6 > len(d.params) {
		return word.Param{}
	}
	p := d.params[i:]
	return word.Param{
		LeftID:  binary.LittleEndian.Uint16(p),
		RightID: binary.LittleEndian.Uint16(p[2:]),
		Cost:    int16(binary.LittleEndian.Uint16(p[4:])),
	}
}

// Info は語の詳細情報を返す。
func (d *Dictionary) Info(id word.ID) (word.Info, error) {
	info, err := d.rawInfo(id.Num())
	if err != nil {
		return word.Info{}, err
	}
	info.DictionaryForm = info.Surface
	if df := info.DictionaryFormWordID; df >= 0 && uint32(df) != id.Num() {
		base, err := d.rawInfo(uint32(df))
		if err != nil {
			return word.Info{}, fmt.Errorf("read dictionary form of %d: %w", id.Num(), err)
		}
		info.DictionaryForm = base.Surface
	}
	return info, nil
}

func (d *Dictionary) rawInfo(num uint32) (word.Info, error) {
	if int(num) >= d.numWords {
		return word.Info{}, fmt.Errorf("word number %d out of range [0, %d)", num, d.numWords)
	}
	off := int(binary.LittleEndian.Uint32(d.infoOffsets[4*num:]))
	r := newReader(d.data, off)
	var info word.Info
	info.Surface = r.utf16String()
	info.HeadWordLength = uint16(r.varLen())
	info.POSID = r.u16()
	info.NormalizedForm = r.utf16String()
	info.DictionaryFormWordID = int32(r.u32())
	info.ReadingForm = r.utf16String()
	info.AUnitSplit = r.u32Array[word.ID]()
	info.BUnitSplit = r.u32Array[word.ID]()
	info.WordStructure = r.u32Array[word.ID]()
	if d.hasSynonymID {
		info.SynonymGroupIDs = r.u32Array[uint32]()
	}
	if r.err != nil {
		return word.Info{}, fmt.Errorf("read word %d: %w", num, r.err)
	}
	if info.NormalizedForm == "" {
		info.NormalizedForm = info.Surface
	}
	if info.ReadingForm == "" {
		info.ReadingForm = info.Surface
	}
	return info, nil
}

// CommonPrefix は b[offset:] の接頭辞に一致する語を短い順に列挙する。
func (d *Dictionary) CommonPrefix(b []byte, offset int) iter.Seq[tokenize.LexiconEntry] {
	return func(yield func(tokenize.LexiconEntry) bool) {
		for end, value := range d.trieCommonPrefix(b, offset) {
			for num := range d.wordIDs(value) {
				if !yield(tokenize.LexiconEntry{WordID: word.ID(num), End: end}) {
					return
				}
			}
		}
	}
}

func (d *Dictionary) wordIDs(index uint32) iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		i := int(index)
		if i >= len(d.wordIDTable) {
			return
		}
		n := int(d.wordIDTable[i])
		ids := d.wordIDTable[i+1:]
		for k := range min(n, len(ids)/4) {
			if !yield(binary.LittleEndian.Uint32(ids[4*k:])) {
				return
			}
		}
	}
}
