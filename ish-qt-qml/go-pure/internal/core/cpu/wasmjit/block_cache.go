package wasmjit

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
)

const cacheVersion = 1

type BlockCache struct{ Dir string }

func NewBlockCache(dir string) (*BlockCache, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &BlockCache{Dir: dir}, nil
}

func KeyForBlock(block GuestBlock) BlockKey {
	h := sha256.New()
	h.Write([]byte(block.Arch))
	var pc [8]byte
	binary.LittleEndian.PutUint64(pc[:], block.PC)
	h.Write(pc[:])
	h.Write(block.Bytes)
	return BlockKey{Hash: sha256.Sum256(h.Sum(nil)), PC: block.PC, Arch: block.Arch, Version: cacheVersion}
}

func (c *BlockCache) Load(key BlockKey) ([]byte, error) { return os.ReadFile(c.path(key)) }

func (c *BlockCache) Store(key BlockKey, data []byte) error {
	tmp, err := os.CreateTemp(c.Dir, ".block-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err = writeTemp(tmp, data); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, c.path(key))
}

func (c *BlockCache) path(key BlockKey) string {
	name := hex.EncodeToString(key.Hash[:]) + ".wasm"
	return filepath.Join(c.Dir, name)
}

func writeTemp(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
