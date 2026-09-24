package sudachin_test

import (
	"errors"
	"testing"

	sudachin "github.com/ZONO33LHD/sudachin-go"
)

func TestAnalyzeAfterClose(t *testing.T) {
	a := openGoldenAnalyzer(t)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Analyze("東京都", sudachin.ModeC); !errors.Is(err, sudachin.ErrClosed) {
		t.Errorf("Analyze after Close error = %v, want ErrClosed", err)
	}
	if err := a.Close(); err != nil {
		t.Errorf("second Close error = %v, want nil", err)
	}
}
