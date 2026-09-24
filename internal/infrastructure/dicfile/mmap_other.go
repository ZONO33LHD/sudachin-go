//go:build !unix

package dicfile

import "os"

// mapFile は mmap を使えない環境向けに、辞書ファイル全体をメモリへ読み込む。
func mapFile(path string) ([]byte, func() error, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return data, func() error { return nil }, nil
}
