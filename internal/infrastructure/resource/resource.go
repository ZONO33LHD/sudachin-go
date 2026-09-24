// Package resource は char.def / unk.def / rewrite.def を解析する。既定の定義ファイルはバイナリに埋め込む。
package resource

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/chars"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

var (
	//go:embed data/char.def
	defaultCharDef []byte
	//go:embed data/unk.def
	defaultUnkDef []byte
	//go:embed data/rewrite.def
	defaultRewriteDef []byte
)

// DefaultCharTable は同梱の char.def から文字種表を作る。
func DefaultCharTable() (*chars.Table, error) { return ParseCharDef(bytes.NewReader(defaultCharDef)) }

// DefaultUnkDef は同梱の unk.def を解析する。
func DefaultUnkDef() ([]UnknownWord, error) { return ParseUnkDef(bytes.NewReader(defaultUnkDef)) }

// DefaultRewriteDef は同梱の rewrite.def を解析する。
func DefaultRewriteDef() (RewriteDef, error) {
	return ParseRewriteDef(bytes.NewReader(defaultRewriteDef))
}

// lines はコメント (#) と空行を除いた行を、行番号付きで fn に渡す。
func lines(r io.Reader, fn func(no int, line string) error) error {
	sc := bufio.NewScanner(r)
	for no := 1; sc.Scan(); no++ {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := fn(no, line); err != nil {
			return fmt.Errorf("line %d: %w", no, err)
		}
	}
	return sc.Err()
}

// ParseCharDef は char.def 形式の定義を解析する。
func ParseCharDef(r io.Reader) (*chars.Table, error) {
	var ranges []chars.Range
	var infos []chars.Info
	err := lines(r, func(_ int, line string) error {
		cols := strings.Fields(line)
		if strings.HasPrefix(cols[0], "0x") {
			rg, err := parseCharRange(cols)
			if err != nil {
				return err
			}
			ranges = append(ranges, rg)
			return nil
		}
		info, err := parseCategoryInfo(cols)
		if err != nil {
			return err
		}
		infos = append(infos, info)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse char.def: %w", err)
	}
	return chars.NewTable(ranges, infos)
}

func parseCharRange(cols []string) (chars.Range, error) {
	if len(cols) < 2 {
		return chars.Range{}, fmt.Errorf("character range needs at least one category: %q", strings.Join(cols, " "))
	}
	lo, hi, found := strings.Cut(cols[0], "..")
	begin, err := strconv.ParseInt(strings.TrimPrefix(lo, "0x"), 16, 32)
	if err != nil {
		return chars.Range{}, fmt.Errorf("invalid code point %q: %w", lo, err)
	}
	end := begin
	if found {
		if end, err = strconv.ParseInt(strings.TrimPrefix(hi, "0x"), 16, 32); err != nil {
			return chars.Range{}, fmt.Errorf("invalid code point %q: %w", hi, err)
		}
	}
	var cat chars.Category
	for _, name := range cols[1:] {
		c, ok := chars.ParseCategory(name)
		if !ok {
			return chars.Range{}, fmt.Errorf("unknown character category %q", name)
		}
		cat |= c
	}
	return chars.Range{Begin: rune(begin), End: rune(end) + 1, Category: cat}, nil
}

func parseCategoryInfo(cols []string) (chars.Info, error) {
	if len(cols) < 4 {
		return chars.Info{}, fmt.Errorf("category definition needs 4 columns: %q", strings.Join(cols, " "))
	}
	c, ok := chars.ParseCategory(cols[0])
	if !ok {
		return chars.Info{}, fmt.Errorf("unknown character category %q", cols[0])
	}
	var nums [3]int
	for i := range nums {
		n, err := strconv.Atoi(cols[i+1])
		if err != nil {
			return chars.Info{}, fmt.Errorf("invalid number %q in category %s: %w", cols[i+1], cols[0], err)
		}
		nums[i] = n
	}
	return chars.Info{Category: c, Invoke: nums[0] != 0, Group: nums[1] != 0, Length: nums[2]}, nil
}

// UnknownWord は unk.def の 1 行 (文字種ごとの未知語の品詞とコスト)。
type UnknownWord struct {
	Category chars.Category
	Param    word.Param
	POS      word.POS
}

// ParseUnkDef は unk.def 形式の定義を解析する。
func ParseUnkDef(r io.Reader) ([]UnknownWord, error) {
	var defs []UnknownWord
	err := lines(r, func(_ int, line string) error {
		cols := strings.Split(line, ",")
		if len(cols) != 4+word.POSDepth {
			return fmt.Errorf("unknown word definition needs %d columns, got %d", 4+word.POSDepth, len(cols))
		}
		c, ok := chars.ParseCategory(cols[0])
		if !ok {
			return fmt.Errorf("unknown character category %q", cols[0])
		}
		var nums [3]int64
		for i := range nums {
			n, err := strconv.ParseInt(strings.TrimSpace(cols[i+1]), 10, 16)
			if err != nil {
				return fmt.Errorf("invalid number %q: %w", cols[i+1], err)
			}
			nums[i] = n
		}
		pos, err := word.ParsePOS(cols[4:])
		if err != nil {
			return err
		}
		defs = append(defs, UnknownWord{
			Category: c,
			Param:    word.Param{LeftID: uint16(nums[0]), RightID: uint16(nums[1]), Cost: int16(nums[2])},
			POS:      pos,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse unk.def: %w", err)
	}
	return defs, nil
}

// RewriteDef は rewrite.def の内容。
type RewriteDef struct {
	// IgnoreNormalize は正規化 (小文字化・NFKC) を行わない文字。
	IgnoreNormalize map[rune]bool
	// Replace は正規化より優先して適用する文字列置換。
	Replace map[string]string
}

// ParseRewriteDef は rewrite.def 形式の定義を解析する。
// 1 列の行は正規化しない文字、2 列の行は置換規則として扱う。
func ParseRewriteDef(r io.Reader) (RewriteDef, error) {
	def := RewriteDef{IgnoreNormalize: map[rune]bool{}, Replace: map[string]string{}}
	err := lines(r, func(_ int, line string) error {
		cols := strings.Fields(line)
		switch len(cols) {
		case 1:
			ch, size := utf8.DecodeRuneInString(cols[0])
			if size != len(cols[0]) {
				return fmt.Errorf("ignore-normalize entry must be a single character: %q", cols[0])
			}
			def.IgnoreNormalize[ch] = true
		case 2:
			if _, dup := def.Replace[cols[0]]; dup {
				return fmt.Errorf("replacement for %q is defined twice", cols[0])
			}
			def.Replace[cols[0]] = cols[1]
		default:
			return fmt.Errorf("expected 1 or 2 columns, got %d", len(cols))
		}
		return nil
	})
	if err != nil {
		return RewriteDef{}, fmt.Errorf("parse rewrite.def: %w", err)
	}
	return def, nil
}
