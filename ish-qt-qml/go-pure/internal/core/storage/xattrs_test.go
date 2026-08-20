package storage

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestXattrCRUD(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	inode, err := db.PathCreate(ctx, "/file", Stat{Mode: 0o100644})
	if err != nil {
		t.Fatal(err)
	}
	value := []byte("value")
	if err := db.SetXattr(ctx, inode, "user.test", value, XattrCreate); err != nil {
		t.Fatal(err)
	}
	value[0] = 'V'
	got, err := db.GetXattr(ctx, inode, "user.test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []byte("value")) {
		t.Fatalf("GetXattr = %q, want value", got)
	}
	if err := db.SetXattr(ctx, inode, "user.test", []byte("new"), XattrCreate); !errors.Is(err, ErrExists) {
		t.Fatalf("create existing error = %v, want ErrExists", err)
	}
	if err := db.SetXattr(ctx, inode, "user.test", []byte("new"), XattrReplace); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetXattr(ctx, inode, "user.test")
	if err != nil || !reflect.DeepEqual(got, []byte("new")) {
		t.Fatalf("replaced value = %q, err=%v", got, err)
	}
	if names, err := db.ListXattrs(ctx, inode); err != nil || string(names) != "user.test\x00" {
		t.Fatalf("ListXattrs = %q, err=%v", names, err)
	}
	if err := db.RemoveXattr(ctx, inode, "user.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetXattr(ctx, inode, "user.test"); !errors.Is(err, ErrNoData) {
		t.Fatalf("missing xattr error = %v, want ErrNoData", err)
	}
}
