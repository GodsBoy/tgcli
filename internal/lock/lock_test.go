package lock

import (
	"testing"
)

func TestAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lk.Release()

	// Second acquire should fail.
	_, err = Acquire(dir)
	if err == nil {
		t.Fatal("expected error on second acquire")
	}

	// Release and re-acquire should work.
	if err := lk.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}

	lk2, err := Acquire(dir)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	lk2.Release()
}
