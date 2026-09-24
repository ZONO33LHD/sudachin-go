package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "dictionary is required", args: []string{}, wantErr: "dictionary is not specified"},
		{name: "invalid mode", args: []string{"-l", "x.dic", "-m", "D"}, wantErr: "split mode"},
		{name: "invalid format", args: []string{"-l", "x.dic", "-f", "xml"}, wantErr: "format must be"},
		{name: "valid", args: []string{"-l", "x.dic", "-m", "a", "-a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SUDACHIN_DICT", "")
			_, err := parseFlags(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseFlags(%v) unexpected error: %v", tt.args, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseFlags(%v) error = %v, want %q", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestRunReportsMissingDictionary(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-l", "testdata/does-not-exist.dic"}, strings.NewReader("東京"), &out)
	if err == nil || !strings.Contains(err.Error(), "does-not-exist.dic") {
		t.Errorf("run() error = %v, want it to mention the dictionary path", err)
	}
}

func TestRunWithDictionary(t *testing.T) {
	dict := os.Getenv("SUDACHIN_TEST_DICT")
	if dict == "" {
		t.Skip("set SUDACHIN_TEST_DICT to run the CLI end to end")
	}
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-m", "A"}, "東京\t名詞,固有名詞,地名,一般,*,*\t東京\n都\t名詞,普通名詞,一般,*,*,*\t都\nEOS\n"},
		{[]string{"-w"}, "東京都\n"},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		if err := run(append([]string{"-l", dict}, tt.args...), strings.NewReader("東京都\n"), &out); err != nil {
			t.Fatal(err)
		}
		if out.String() != tt.want {
			t.Errorf("run(%v) output = %q, want %q", tt.args, out.String(), tt.want)
		}
	}
}

func TestRunWritesOutputFile(t *testing.T) {
	dict := os.Getenv("SUDACHIN_TEST_DICT")
	if dict == "" {
		t.Skip("set SUDACHIN_TEST_DICT to run the CLI end to end")
	}
	out := t.TempDir() + "/out.txt"
	if err := run([]string{"-l", dict, "-w", "-o", out}, strings.NewReader("東京都\n"), io.Discard); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "東京都\n" {
		t.Errorf("output file = %q", b)
	}
}
