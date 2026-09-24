package pathrewrite

import (
	"strings"
	"testing"
)

func TestNumberParser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    string
		wantOK  bool
		wantErr parseError
	}{
		{in: "1000", want: "1000", wantOK: true},
		{in: "一千", want: "1000", wantOK: true},
		{in: "千", want: "1000", wantOK: true},
		{in: "一万二千三百四十五", want: "12345", wantOK: true},
		{in: "三億五千万", want: "350000000", wantOK: true},
		{in: "12,300", want: "12300", wantOK: true},
		{in: "1,234,567", want: "1234567", wantOK: true},
		{in: "3.14", want: "3.14", wantOK: true},
		{in: "0.5", want: "0.5", wantOK: true},
		{in: "1.5万", want: "15000", wantOK: true},
		{in: "二〇二六", want: "2026", wantOK: true},
		{in: "12,34", wantOK: false, wantErr: errComma},
		{in: "1.2.3", wantOK: false, wantErr: errPoint},
		{in: "12.", wantOK: false, wantErr: errPoint},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			p := newNumberParser()
			ok := true
			for _, c := range tt.in {
				if !p.append(c) {
					ok = false
					break
				}
			}
			ok = ok && p.done()
			if ok != tt.wantOK {
				t.Fatalf("parse(%q) ok = %v, want %v (err state %d)", tt.in, ok, tt.wantOK, p.err)
			}
			if !ok {
				if p.err != tt.wantErr {
					t.Errorf("parse(%q) error state = %d, want %d", tt.in, p.err, tt.wantErr)
				}
				return
			}
			if got := p.normalized(); got != tt.want {
				t.Errorf("parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// 長い数字列でも 1 桁ごとに数字列全体をコピーしないこと (以前は 80 万桁で 17 秒かかった)。
func TestNumberParserLongInputAllocations(t *testing.T) {
	digits := strings.Repeat("1", 100_000)
	allocs := testing.AllocsPerRun(3, func() {
		p := newNumberParser()
		for _, c := range digits {
			p.append(c)
		}
		if !p.done() || p.normalized() != digits {
			t.Fatal("failed to parse a long digit string")
		}
	})
	if allocs > 200 {
		t.Errorf("parsing 100,000 digits allocated %.0f times; digits are probably copied per append", allocs)
	}
}
