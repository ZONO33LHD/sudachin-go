//go:build unix

package dicfile

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// mapFile は辞書ファイルを読み取り専用で mmap する。数百 MB の辞書をヒープに載せずに済む。
// ファイルを閉じてもマッピングは有効なままなので、ファイルはこの関数内で閉じる。
func mapFile(path string) (data []byte, release func() error, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		cerr := f.Close()
		if cerr == nil {
			return
		}
		if data != nil {
			cerr = errors.Join(cerr, syscall.Munmap(data))
		}
		data, release, err = nil, nil, errors.Join(err, cerr)
	}()
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
	data, err = syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: %w", err)
	}
	mapped := data
	return mapped, func() error { return syscall.Munmap(mapped) }, nil
}
