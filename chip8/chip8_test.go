package chip8

import "testing"

// TestLoad exercises the contract of Machine.Load: where it writes, what it
// leaves untouched, and how it handles ROMs at and beyond capacity.
func TestLoad(t *testing.T) {
	t.Run("places ROM at ProgramStart", func(t *testing.T) {
		m := New()
		rom := []byte{0x12, 0x34, 0xAB, 0xCD}

		if err := m.Load(rom); err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}

		for i, want := range rom {
			if got := m.Memory[ProgramStart+i]; got != want {
				t.Errorf("Memory[0x%X] = 0x%02X, want 0x%02X", ProgramStart+i, got, want)
			}
		}

		// The byte just past the ROM should be untouched (still zero).
		if got := m.Memory[ProgramStart+len(rom)]; got != 0 {
			t.Errorf("Memory[0x%X] = 0x%02X, want 0x00 (past end of ROM)", ProgramStart+len(rom), got)
		}
	})

	t.Run("leaves fontset intact", func(t *testing.T) {
		m := New()
		if err := m.Load([]byte{0xFF, 0xFF, 0xFF}); err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}

		for i, want := range fontSet {
			if got := m.Memory[FontStart+i]; got != want {
				t.Errorf("font byte at Memory[0x%X] = 0x%02X, want 0x%02X", FontStart+i, got, want)
			}
		}
	})

	t.Run("rejects oversized ROM", func(t *testing.T) {
		m := New()
		rom := make([]byte, MaxROMSize+1)

		if err := m.Load(rom); err == nil {
			t.Fatalf("Load(%d bytes) returned nil error, want an error", len(rom))
		}
	})

	t.Run("accepts max-size ROM", func(t *testing.T) {
		m := New()
		rom := make([]byte, MaxROMSize)

		if err := m.Load(rom); err != nil {
			t.Fatalf("Load(%d bytes) returned error %v, want nil at exact capacity", len(rom), err)
		}
	})
}
