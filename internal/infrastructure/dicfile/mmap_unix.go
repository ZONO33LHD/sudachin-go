//go:build unix

package dicfile

import (
	"fmt"
	"os"
	"syscall"
)

// mapFile は辞書ファイルを読み取り専用で mmap する。数百 MB の辞書をヒープに載せずに済む。
func mapFile(path string) ([]byte, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	size := st.Size()
	if size == 0 {
		return nil, nil, fmt.Errorf("%s is empty", path)
	}
	if int64(int(size)) != size {
		return nil, nil, fmt.Errorf("%s is too large to map (%d bytes)", path, size)
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: %w", err)
	}
	return data, func() error { return syscall.Munmap(data) }, nil
}
