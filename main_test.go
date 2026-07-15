package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chip8-emu/chip8"
)

func TestRun(t *testing.T) {
	// The -tui flag keeps run on the terminal path so tests never open a window.
	t.Run("loads and runs a valid ROM without error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "valid.ch8")
		// 00E0 (clear), then 1202: jump to self at 0x202 so execution settles.
		rom := []byte{0x00, 0xE0, 0x12, 0x02}
		if err := os.WriteFile(path, rom, 0o600); err != nil {
			t.Fatalf("writing temp ROM: %v", err)
		}

		if err := run([]string{"-tui", path}, io.Discard); err != nil {
			t.Errorf("run(%q) = %v, want nil", path, err)
		}
	})

	t.Run("rejects an oversized ROM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "big.ch8")
		if err := os.WriteFile(path, make([]byte, chip8.MaxROMSize+1), 0o600); err != nil {
			t.Fatalf("writing temp ROM: %v", err)
		}

		if err := run([]string{"-tui", path}, io.Discard); err == nil {
			t.Error("run on oversized ROM returned nil, want an error")
		}
	})

	t.Run("errors when no argument is given", func(t *testing.T) {
		if err := run(nil, io.Discard); err == nil {
			t.Error("run(nil) returned nil, want a usage error")
		}
	})

	t.Run("errors when too many arguments are given", func(t *testing.T) {
		if err := run([]string{"-tui", "a.ch8", "b.ch8"}, io.Discard); err == nil {
			t.Error("run with two args returned nil, want a usage error")
		}
	})

	t.Run("errors when the file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nope.ch8") // never created
		if err := run([]string{"-tui", path}, io.Discard); err == nil {
			t.Errorf("run on missing file returned nil, want an error")
		}
	})
}

func TestExecute(t *testing.T) {
	t.Run("runs until a self-loop then renders", func(t *testing.T) {
		m := chip8.New()
		// A300: I = 0x300; D011: draw a 1-row sprite at (V0,V1)=(0,0);
		// 1204: jump to self at 0x204 (settles).
		if err := m.Load([]byte{0xA3, 0x00, 0xD0, 0x11, 0x12, 0x04}); err != nil {
			t.Fatalf("Load: %v", err)
		}
		m.Memory[0x300] = 0x80 // single lit pixel, top-left of the sprite

		var out strings.Builder
		if err := execute(m, &out); err != nil {
			t.Fatalf("execute: %v", err)
		}
		if !m.Display[0][0] {
			t.Error("Display[0][0] not set; the sprite should have been drawn")
		}
		if []rune(strings.SplitN(out.String(), "\n", 2)[0])[0] != onPixel {
			t.Error("rendered output does not show a lit top-left pixel")
		}
	})

	t.Run("propagates opcode errors", func(t *testing.T) {
		m := chip8.New()
		if err := m.Load([]byte{0x50, 0x00}); err != nil { // 5XY0: unimplemented
			t.Fatalf("Load: %v", err)
		}
		if err := execute(m, io.Discard); err == nil {
			t.Error("execute returned nil for an unimplemented opcode, want an error")
		}
	})
}

func TestRender(t *testing.T) {
	m := chip8.New()
	m.Display[0][0] = true                                        // top-left
	m.Display[chip8.DisplayWidth-1][chip8.DisplayHeight-1] = true // bottom-right

	lines := strings.Split(strings.TrimRight(render(m), "\n"), "\n")

	if len(lines) != chip8.DisplayHeight {
		t.Fatalf("rendered %d lines, want %d", len(lines), chip8.DisplayHeight)
	}
	if got := []rune(lines[0]); len(got) != chip8.DisplayWidth {
		t.Fatalf("row 0 has %d runes, want %d", len(got), chip8.DisplayWidth)
	}
	if r := []rune(lines[0])[0]; r != onPixel {
		t.Errorf("top-left rune = %q, want onPixel %q", r, onPixel)
	}
	if r := []rune(lines[0])[1]; r != offPixel {
		t.Errorf("second rune of row 0 = %q, want offPixel %q", r, offPixel)
	}
	if r := []rune(lines[chip8.DisplayHeight-1])[chip8.DisplayWidth-1]; r != onPixel {
		t.Errorf("bottom-right rune = %q, want onPixel %q", r, onPixel)
	}
}
