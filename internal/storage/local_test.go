package storage

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

const testKey = "12345678-1234-4234-8234-123456789abc.png"

func TestLocalCreateOpenAndRemove(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("test attachment")
	if err := store.Create(testKey, data); err != nil {
		t.Fatal(err)
	}
	file, err := store.Open(testKey)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || string(got) != string(data) {
		t.Fatalf("read attachment = %q, read error = %v, close error = %v", got, readErr, closeErr)
	}
	if err := store.Remove(testKey); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.root, testKey)); !os.IsNotExist(err) {
		t.Fatalf("attachment still exists or stat failed: %v", err)
	}
}

func TestLocalRejectsUnsafeKeysAndDuplicateFiles(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(testKey, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(testKey, []byte("second")); err == nil {
		t.Fatal("Create overwrote an existing attachment")
	}
	for _, key := range []string{"../secret.png", "not-a-uuid.jpg"} {
		if _, err := store.Open(key); err == nil {
			t.Errorf("Open accepted unsafe key %q", key)
		}
		if err := store.Remove(key); err == nil {
			t.Errorf("Remove accepted unsafe key %q", key)
		}
	}
}
