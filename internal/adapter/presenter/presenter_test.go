package presenter

import (
	"bytes"
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/morpheme"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

func TestWriter(t *testing.T) {
	t.Parallel()
	ms := []morpheme.Morpheme{
		{Surface: "東京都", POS: word.POS{"名詞", "固有名詞", "地名", "一般", "*", "*"}, NormalizedForm: "東京都", DictionaryForm: "東京都", ReadingForm: "トウキョウト", SynonymGroupIDs: []uint32{1, 2}, End: 9},
		{Surface: "ＡＢ", POS: word.POS{"名詞", "普通名詞", "一般", "*", "*", "*"}, NormalizedForm: "ab", DictionaryForm: "ab", ReadingForm: "ab", DictionaryID: -1, OOV: true, Begin: 9, End: 15},
	}
	tests := []struct {
		format Format
		want   string
	}{
		{Simple, "東京都\t名詞,固有名詞,地名,一般,*,*\t東京都\nＡＢ\t名詞,普通名詞,一般,*,*,*\tab\nEOS\n"},
		{All, "東京都\t名詞,固有名詞,地名,一般,*,*\t東京都\t東京都\tトウキョウト\t0\t[1, 2]\nＡＢ\t名詞,普通名詞,一般,*,*,*\tab\tab\tab\t-1\t[]\t(OOV)\nEOS\n"},
		{Wakati, "東京都 ＡＢ\n"},
		{JSON, `[{"surface":"東京都","pos":["名詞","固有名詞","地名","一般","*","*"],"normalized_form":"東京都","dictionary_form":"東京都","reading_form":"トウキョウト","dictionary_id":0,"synonym_group_ids":[1,2],"is_oov":false,"begin":0,"end":9},{"surface":"ＡＢ","pos":["名詞","普通名詞","一般","*","*","*"],"normalized_form":"ab","dictionary_form":"ab","reading_form":"ab","dictionary_id":-1,"synonym_group_ids":[],"is_oov":true,"begin":9,"end":15}]` + "\n"},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		w := NewWriter(&buf, tt.format)
		if err := w.Write(ms); err != nil {
			t.Fatal(err)
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
		if got := buf.String(); got != tt.want {
			t.Errorf("format %d:\n got %q\nwant %q", tt.format, got, tt.want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()
	if f, err := ParseFormat("json"); err != nil || f != JSON {
		t.Errorf("ParseFormat(json) = %v, %v", f, err)
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml) should fail")
	}
}
