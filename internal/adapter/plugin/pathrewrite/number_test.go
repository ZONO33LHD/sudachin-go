package pathrewrite

import "testing"

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
