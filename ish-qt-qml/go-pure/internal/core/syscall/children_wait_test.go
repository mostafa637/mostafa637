package syscall

import (
	"testing"
	"time"
)

func TestChildRegistryBlockingWaitWakesOnExit(t *testing.T) {
	registry := NewChildRegistry()
	if !registry.AddChild(100, 200) {
		t.Fatal("add child failed")
	}
	started := make(chan struct{})
	result := make(chan struct {
		pid    int32
		status int32
		err    int32
	}, 1)
	go func() {
		close(started)
		pid, status, err := registry.Wait(100, 200, 0)
		result <- struct {
			pid    int32
			status int32
			err    int32
		}{pid, status, err}
	}()
	<-started
	select {
	case got := <-result:
		t.Fatalf("wait returned before child exit: %+v", got)
	case <-time.After(20 * time.Millisecond):
	}
	if !registry.MarkExited(200, 7) {
		t.Fatal("mark exited failed")
	}
	select {
	case got := <-result:
		if got.err != 0 || got.pid != 200 || got.status != 7<<8 {
			t.Fatalf("wait result: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("blocking wait was not woken")
	}
}

func TestChildRegistryWaitIDNoWaitKeepsChild(t *testing.T) {
	registry := NewChildRegistry()
	if !registry.AddChild(100, 201) || !registry.MarkExited(201, 9) {
		t.Fatal("child setup failed")
	}
	pid, code, exited, err := registry.WaitID(100, waitIDTypePID64, 201, WaitExited|WaitNoWait)
	if err != 0 || pid != 201 || code != 9 || !exited {
		t.Fatalf("first waitid: pid=%d code=%d exited=%v err=%d", pid, code, exited, err)
	}
	pid, code, exited, err = registry.WaitID(100, waitIDTypePID64, 201, WaitExited)
	if err != 0 || pid != 201 || code != 9 || !exited {
		t.Fatalf("second waitid: pid=%d code=%d exited=%v err=%d", pid, code, exited, err)
	}
	if _, _, _, err = registry.WaitID(100, waitIDTypePID64, 201, WaitExited|WaitNoHang); err != ECHILD {
		t.Fatalf("reaped waitid err=%d", err)
	}
}

func TestChildRegistryBlockingWaitIDWakesOnExit(t *testing.T) {
	registry := NewChildRegistry()
	if !registry.AddChild(100, 202) {
		t.Fatal("add child failed")
	}
	result := make(chan struct {
		pid    uint32
		code   int32
		exited bool
		err    int32
	}, 1)
	go func() {
		pid, code, exited, err := registry.WaitID(100, waitIDTypePID64, 202, WaitExited)
		result <- struct {
			pid    uint32
			code   int32
			exited bool
			err    int32
		}{pid, code, exited, err}
	}()
	select {
	case got := <-result:
		t.Fatalf("waitid returned before child exit: %+v", got)
	case <-time.After(20 * time.Millisecond):
	}
	if !registry.MarkExited(202, 11) {
		t.Fatal("mark exited failed")
	}
	select {
	case got := <-result:
		if got.err != 0 || got.pid != 202 || got.code != 11 || !got.exited {
			t.Fatalf("waitid result: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("blocking waitid was not woken")
	}
}
