package dicfile

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

const testDict = "../../../testdata/system.dic.test"

func openTestDict(t *testing.T) *Dictionary {
	t.Helper()
	d, err := Open(testDict)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func TestOpen(t *testing.T) {
	t.Parallel()
	d := openTestDict(t)
	if d.Header.Description != "the system dictionary for the unit tests" {
		t.Errorf("Description = %q", d.Header.Description)
	}
	if d.numWords != 39 {
		t.Errorf("numWords = %d, want 39", d.numWords)
	}
	p, ok := d.POS(3)
	if !ok || p.String() != "名詞,固有名詞,地名,一般,*,*" {
		t.Errorf("POS(3) = %v, %v", p, ok)
	}
	if id, ok := d.POSID(p); !ok || id != 3 {
		t.Errorf("POSID(%v) = %d, %v, want 3", p, id, ok)
	}
	if _, ok := d.POS(100); ok {
		t.Error("POS(100) should not exist")
	}
}

func TestCommonPrefix(t *testing.T) {
	t.Parallel()
	d := openTestDict(t)
	input := []byte("東京都に")
	var got []string
	for e := range d.CommonPrefix(input, 0) {
		info, err := d.Info(e.WordID)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, string(input[:e.End])+"="+info.Surface)
	}
	want := []string{"東=東", "東京=東京", "東京都=東京都"}
	if !slices.Equal(got, want) {
		t.Errorf("CommonPrefix(東京都に) = %v, want %v", got, want)
	}

	// offset 以降だけを検索し、End は入力全体に対する位置になる
	for e := range d.CommonPrefix(input, len("東京都")) {
		if e.End != len(input) {
			t.Errorf("entry from offset ends at %d, want %d", e.End, len(input))
		}
	}
}

func TestInfo(t *testing.T) {
	t.Parallel()
	d := openTestDict(t)
	info, err := d.Info(word.ID(6))
	if err != nil {
		t.Fatal(err)
	}
	if info.Surface != "東京都" || info.ReadingForm == "" || info.DictionaryForm != "東京都" {
		t.Errorf("Info(6) = %+v", info)
	}
	if !slices.Equal(info.AUnitSplit, []word.ID{5, 9}) {
		t.Errorf("AUnitSplit = %v, want [5 9]", info.AUnitSplit)
	}
	if p := d.Param(word.ID(6)); p != (word.Param{LeftID: 6, RightID: 8, Cost: 5320}) {
		t.Errorf("Param(6) = %+v", p)
	}
	if _, err := d.Info(word.ID(1000)); err == nil {
		t.Error("Info(1000) should fail")
	}
}

func TestParseRejectsBrokenData(t *testing.T) {
	t.Parallel()
	d := openTestDict(t)
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{name: "empty", data: nil, wantErr: "truncated"},
		{name: "unknown version", data: make([]byte, 300), wantErr: "unsupported dictionary version"},
		{name: "truncated lexicon", data: d.data[:len(d.data)/2], wantErr: "truncated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(tt.data); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	seed, err := os.ReadFile(testDict)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := Parse(data)
		if err != nil {
			return
		}
		for e := range d.CommonPrefix([]byte("東京都"), 0) {
			_, _ = d.Info(e.WordID)
		}
	})
}
