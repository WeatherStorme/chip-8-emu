package main

import (
	"os"
	"path/filepath"
	"testing"

	"chip8-emu/chip8"
)

func TestRun(t *testing.T) {
	t.Run("loads a valid ROM without error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "valid.ch8")
		// A few real opcodes: 00E0, A22A, D015, 1200.
		rom := []byte{0x00, 0xE0, 0xA2, 0x2A, 0xD0, 0x15, 0x12, 0x00}
		if err := os.WriteFile(path, rom, 0o600); err != nil {
			t.Fatalf("writing temp ROM: %v", err)
		}

		if err := run([]string{path}); err != nil {
			t.Errorf("run(%q) = %v, want nil", path, err)
		}
	})

	t.Run("rejects an oversized ROM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "big.ch8")
		if err := os.WriteFile(path, make([]byte, chip8.MaxROMSize+1), 0o600); err != nil {
			t.Fatalf("writing temp ROM: %v", err)
		}

		if err := run([]string{path}); err == nil {
			t.Error("run on oversized ROM returned nil, want an error")
		}
	})

	t.Run("errors when no argument is given", func(t *testing.T) {
		if err := run(nil); err == nil {
			t.Error("run(nil) returned nil, want a usage error")
		}
	})

	t.Run("errors when too many arguments are given", func(t *testing.T) {
		if err := run([]string{"a.ch8", "b.ch8"}); err == nil {
			t.Error("run with two args returned nil, want a usage error")
		}
	})

	t.Run("errors when the file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nope.ch8") // never created
		if err := run([]string{path}); err == nil {
			t.Errorf("run on missing file returned nil, want an error")
		}
	})
}
