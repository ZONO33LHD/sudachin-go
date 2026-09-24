// Package presenter は解析結果を CLI 向けの文字列形式に変換する。
package presenter

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/morpheme"
)

// Format は出力形式。
type Format int

const (
	// Simple は「表層形 品詞 正規化形」を 1 行ずつ出し、文末に EOS を出す (Sudachi 既定)。
	Simple Format = iota
	// All は Simple に辞書形・読み・辞書番号・同義語グループ ID・未知語フラグを加える (sudachi -a)。
	All
	// Wakati は表層形を空白区切りで 1 行に出す。
	Wakati
	// JSON は 1 文を 1 行の JSON 配列で出す。
	JSON
)

// ParseFormat は "simple" / "all" / "wakati" / "json" を Format に変換する。
func ParseFormat(s string) (Format, error) {
	switch s {
	case "simple":
		return Simple, nil
	case "all":
		return All, nil
	case "wakati":
		return Wakati, nil
	case "json":
		return JSON, nil
	default:
		return 0, fmt.Errorf(`format must be one of "simple", "all", "wakati", "json", got %q`, s)
	}
}

// Writer は 1 文ずつ解析結果を書き出す。
type Writer struct {
	w      *bufio.Writer
	json   *jsontext.Encoder
	format Format
}

// NewWriter は w に format 形式で書き出す Writer を作る。書き終えたら Flush すること。
func NewWriter(w io.Writer, format Format) *Writer {
	bw := bufio.NewWriter(w)
	return &Writer{w: bw, json: jsontext.NewEncoder(bw), format: format}
}

// Flush はバッファを書き出す。
func (w *Writer) Flush() error { return w.w.Flush() }

// Write は 1 文分の形態素を書き出す。
func (w *Writer) Write(ms []morpheme.Morpheme) error {
	switch w.format {
	case Wakati:
		surfaces := make([]string, len(ms))
		for i, m := range ms {
			surfaces[i] = m.Surface
		}
		_, err := fmt.Fprintln(w.w, strings.Join(surfaces, " "))
		return err
	case JSON:
		// jsontext.Encoder はトップレベルの値ごとに改行を付けるので、1 文が 1 行になる。
		return json.MarshalEncode(w.json, toJSON(ms))
	default:
		for _, m := range ms {
			w.writeLine(m)
		}
		_, err := w.w.WriteString("EOS\n")
		return err
	}
}

func (w *Writer) writeLine(m morpheme.Morpheme) {
	fields := []string{m.Surface, m.POS.String(), m.NormalizedForm}
	if w.format == All {
		fields = append(fields, m.DictionaryForm, m.ReadingForm, strconv.Itoa(m.DictionaryID), formatIDs(m.SynonymGroupIDs))
		if m.OOV {
			fields = append(fields, "(OOV)")
		}
	}
	w.w.WriteString(strings.Join(fields, "\t"))
	w.w.WriteByte('\n')
}

func formatIDs(ids []uint32) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(uint64(id), 10)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// jsonMorpheme は JSON 出力の 1 要素。encoding/json/v2 は nil スライスも [] として出力する。
type jsonMorpheme struct {
	Surface         string   `json:"surface"`
	POS             []string `json:"pos"`
	NormalizedForm  string   `json:"normalized_form"`
	DictionaryForm  string   `json:"dictionary_form"`
	ReadingForm     string   `json:"reading_form"`
	DictionaryID    int      `json:"dictionary_id"`
	SynonymGroupIDs []uint32 `json:"synonym_group_ids"`
	OOV             bool     `json:"is_oov"`
	Begin           int      `json:"begin"`
	End             int      `json:"end"`
}

func toJSON(ms []morpheme.Morpheme) []jsonMorpheme {
	out := make([]jsonMorpheme, len(ms))
	for i, m := range ms {
		out[i] = jsonMorpheme{
			Surface: m.Surface, POS: m.POS[:], NormalizedForm: m.NormalizedForm,
			DictionaryForm: m.DictionaryForm, ReadingForm: m.ReadingForm, DictionaryID: m.DictionaryID,
			SynonymGroupIDs: m.SynonymGroupIDs, OOV: m.OOV, Begin: m.Begin, End: m.End,
		}
	}
	return out
}
