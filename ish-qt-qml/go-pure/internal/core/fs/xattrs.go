package fs

import (
	"context"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func (f *FS) xattrInode(name string, noFollow bool) (uint64, error) {
	if f == nil || f.db == nil {
		return 0, storage.ErrInvariant
	}
	var info FileInfo
	var err error
	if noFollow {
		info, err = f.stat(name, true)
	} else {
		info, err = f.Stat(name)
	}
	if err != nil {
		return 0, err
	}
	if info.Inode == 0 {
		return 0, storage.ErrInvariant
	}
	return info.Inode, nil
}

func (f *FS) SetXattr(name, attr string, value []byte, flags uint32, noFollow bool) error {
	inode, err := f.xattrInode(name, noFollow)
	if err != nil {
		return err
	}
	return f.db.SetXattr(context.Background(), inode, attr, value, flags)
}

func (f *FS) GetXattr(name, attr string, noFollow bool) ([]byte, error) {
	inode, err := f.xattrInode(name, noFollow)
	if err != nil {
		return nil, err
	}
	return f.db.GetXattr(context.Background(), inode, attr)
}

func (f *FS) ListXattr(name string, noFollow bool) ([]byte, error) {
	inode, err := f.xattrInode(name, noFollow)
	if err != nil {
		return nil, err
	}
	return f.db.ListXattrs(context.Background(), inode)
}

func (f *FS) RemoveXattr(name, attr string, noFollow bool) error {
	inode, err := f.xattrInode(name, noFollow)
	if err != nil {
		return err
	}
	return f.db.RemoveXattr(context.Background(), inode, attr)
}

func (f *FS) SetXattrByInode(inode uint64, attr string, value []byte, flags uint32) error {
	if f == nil || f.db == nil {
		return storage.ErrInvariant
	}
	return f.db.SetXattr(context.Background(), inode, attr, value, flags)
}

func (f *FS) GetXattrByInode(inode uint64, attr string) ([]byte, error) {
	if f == nil || f.db == nil {
		return nil, storage.ErrInvariant
	}
	return f.db.GetXattr(context.Background(), inode, attr)
}

func (f *FS) ListXattrByInode(inode uint64) ([]byte, error) {
	if f == nil || f.db == nil {
		return nil, storage.ErrInvariant
	}
	return f.db.ListXattrs(context.Background(), inode)
}

func (f *FS) RemoveXattrByInode(inode uint64, attr string) error {
	if f == nil || f.db == nil {
		return storage.ErrInvariant
	}
	return f.db.RemoveXattr(context.Background(), inode, attr)
}
