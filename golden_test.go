package sudachin_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	sudachin "github.com/ZONO33LHD/sudachin-go"
	"github.com/ZONO33LHD/sudachin-go/internal/adapter/presenter"
)

// testdata/golden の期待値は Sudachi (Rust 版, sudachi.rs) で `sudachi -a --split-sentences no` を実行した出力。
// 辞書の版によって結果が変わるため、生成に使った SudachiDict core 20250515 の system_core.dic を
// 環境変数 SUDACHIN_TEST_DICT で指定したときだけ実行する。
const goldenDictEnv = "SUDACHIN_TEST_DICT"

func openGoldenAnalyzer(tb testing.TB) *sudachin.Analyzer {
	tb.Helper()
	path := os.Getenv(goldenDictEnv)
	if path == "" {
		tb.Skipf("set %s to the SudachiDict core 20250515 system_core.dic to run this test", goldenDictEnv)
	}
	a, err := sudachin.Open(path)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = a.Close() })
	return a
}

func readGolden(t *testing.T, name string) []string {
	t.Helper()
	f, err := os.Open("testdata/golden/" + name + ".txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return strings.SplitAfter(string(b), "EOS\n")
}

func TestGoldenCompatibleWithSudachiRs(t *testing.T) {
	a := openGoldenAnalyzer(t)
	for _, corpus := range []string{"bocchan", "akairousokutoningyo"} {
		input, err := os.ReadFile("testdata/golden/" + corpus + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(string(input), "\n"), "\n")
		for _, mode := range []sudachin.Mode{sudachin.ModeA, sudachin.ModeB, sudachin.ModeC} {
			t.Run(fmt.Sprintf("%s/%v", corpus, mode), func(t *testing.T) {
				want := readGolden(t, fmt.Sprintf("%s_%v", corpus, mode))
				mismatches := 0
				for i, line := range lines {
					ms, err := a.Analyze(line, mode)
					if err != nil {
						t.Fatalf("line %d: %v", i+1, err)
					}
					var buf bytes.Buffer
					w := presenter.NewWriter(&buf, presenter.All)
					if err := w.Write(ms); err != nil {
						t.Fatal(err)
					}
					if err := w.Flush(); err != nil {
						t.Fatal(err)
					}
					if got := buf.String(); i >= len(want) || got != want[i] {
						mismatches++
						if mismatches <= 3 {
							t.Errorf("line %d: %q\n got:\n%s\nwant:\n%s", i+1, line, got, want[min(i, len(want)-1)])
						}
					}
				}
				if mismatches > 0 {
					t.Errorf("%d of %d sentences differ from sudachi.rs", mismatches, len(lines))
				}
			})
		}
	}
}

func BenchmarkAnalyzeBocchan(b *testing.B) {
	a := openGoldenAnalyzer(b)
	f, err := os.Open("testdata/golden/bocchan.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	var lines []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		lines = append(lines, sc.Text())
	}
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			if _, err := a.Analyze(line, sudachin.ModeC); err != nil {
				b.Fatal(err)
			}
		}
	}
}
