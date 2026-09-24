// Command sudachin は標準入力 (またはファイル) の各行を形態素解析して出力する。
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	sudachin "github.com/ZONO33LHD/sudachin-go"
	"github.com/ZONO33LHD/sudachin-go/internal/adapter/presenter"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "sudachin:", err)
		os.Exit(1)
	}
}

type config struct {
	dictPath string
	mode     sudachin.Mode
	format   presenter.Format
	output   string
	inputs   []string
}

func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("sudachin", flag.ContinueOnError)
	dict := fs.String("l", os.Getenv("SUDACHIN_DICT"), "path to the Sudachi system dictionary (default: $SUDACHIN_DICT)")
	mode := fs.String("m", "C", `split mode: "A" (short), "B" (middle) or "C" (named entity)`)
	all := fs.Bool("a", false, "print all fields (same as -f all)")
	wakati := fs.Bool("w", false, "print surfaces only (same as -f wakati)")
	format := fs.String("f", "simple", `output format: "simple", "all", "wakati" or "json"`)
	output := fs.String("o", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	var cfg config
	var err error
	if cfg.mode, err = sudachin.ParseMode(*mode); err != nil {
		return config{}, err
	}
	switch {
	case *all:
		cfg.format = presenter.All
	case *wakati:
		cfg.format = presenter.Wakati
	default:
		if cfg.format, err = presenter.ParseFormat(*format); err != nil {
			return config{}, err
		}
	}
	if *dict == "" {
		return config{}, errors.New("dictionary is not specified: use -l or set SUDACHIN_DICT")
	}
	cfg.dictPath, cfg.output, cfg.inputs = *dict, *output, fs.Args()
	return cfg, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) (err error) {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	a, err := sudachin.Open(cfg.dictPath)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, a.Close()) }()

	if cfg.output != "" {
		f, createErr := os.Create(cfg.output)
		if createErr != nil {
			return createErr
		}
		defer func() { err = errors.Join(err, f.Close()) }()
		stdout = f
	}
	w := presenter.NewWriter(stdout, cfg.format)

	if len(cfg.inputs) == 0 {
		err = analyzeLines(a, stdin, w, cfg.mode)
	}
	for _, path := range cfg.inputs {
		if err = analyzeFile(a, path, w, cfg.mode); err != nil {
			break
		}
	}
	return errors.Join(err, w.Flush())
}

func analyzeFile(a *sudachin.Analyzer, path string, w *presenter.Writer, mode sudachin.Mode) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return analyzeLines(a, f, w, mode)
}

func analyzeLines(a *sudachin.Analyzer, r io.Reader, w *presenter.Writer, mode sudachin.Mode) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for no := 1; sc.Scan(); no++ {
		ms, err := a.Analyze(sc.Text(), mode)
		if err != nil {
			return fmt.Errorf("line %d: %w", no, err)
		}
		if err := w.Write(ms); err != nil {
			return err
		}
	}
	return sc.Err()
}
