package word

import "testing"

func TestNewID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		dic     uint8
		num     uint32
		wantErr bool
	}{
		{name: "system dictionary", dic: 0, num: 12345},
		{name: "last user dictionary", dic: 14, num: MaxWordNum},
		{name: "dictionary id reserved for OOV", dic: 15, num: 1, wantErr: true},
		{name: "word number overflow", dic: 0, num: MaxWordNum + 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := NewID(tt.dic, tt.num)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewID(%d, %d) = %v, want error", tt.dic, tt.num, id)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewID(%d, %d) unexpected error: %v", tt.dic, tt.num, err)
			}
			if id.Dic() != tt.dic || id.Num() != tt.num || id.IsOOV() {
				t.Errorf("NewID(%d, %d) = dic %d num %d oov %v", tt.dic, tt.num, id.Dic(), id.Num(), id.IsOOV())
			}
		})
	}
}

func TestOOVID(t *testing.T) {
	t.Parallel()
	id := OOVID(42)
	if !id.IsOOV() || id.Num() != 42 {
		t.Errorf("OOVID(42) = oov %v num %d, want oov true num 42", id.IsOOV(), id.Num())
	}
	if !InvalidID.IsOOV() {
		t.Error("InvalidID should be treated as OOV")
	}
}

func TestParsePOS(t *testing.T) {
	t.Parallel()
	p, err := ParsePOS([]string{"名詞", "普通名詞", "一般", "*", "*", "*"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := p.String(), "名詞,普通名詞,一般,*,*,*"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if _, err := ParsePOS([]string{"名詞"}); err == nil {
		t.Error("ParsePOS with 1 element should fail")
	}
}
