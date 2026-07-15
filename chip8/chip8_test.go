package chip8

import "testing"

// loadOpcodes returns a fresh Machine with the given opcodes loaded at
// ProgramStart, ready to Step through. Each opcode is written big-endian.
func loadOpcodes(t *testing.T, ops ...uint16) *Machine {
	t.Helper()
	m := New()
	rom := make([]byte, 0, len(ops)*2)
	for _, op := range ops {
		rom = append(rom, byte(op>>8), byte(op))
	}
	if err := m.Load(rom); err != nil {
		t.Fatalf("loading opcodes: %v", err)
	}
	return m
}

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

// TestStep covers fetch/decode plus the simple (non-drawing) opcodes currently
// implemented. DXYN is exercised separately in TestDXYN.
func TestStep(t *testing.T) {
	t.Run("advances PC by two", func(t *testing.T) {
		m := loadOpcodes(t, 0x6100) // 6100: set V1 = 0x00 (a non-jump op)
		start := m.PC
		if start != ProgramStart {
			t.Fatalf("precondition: PC = 0x%03X, want ProgramStart (0x%03X)", start, ProgramStart)
		}
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if m.PC != start+2 { // assert the delta, not an absolute address
			t.Errorf("PC = 0x%03X, want start+2 (0x%03X)", m.PC, start+2)
		}
	})

	t.Run("1NNN jumps to NNN", func(t *testing.T) {
		m := loadOpcodes(t, 0x1234)
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if m.PC != 0x234 {
			t.Errorf("PC = 0x%03X, want 0x234", m.PC)
		}
	})

	t.Run("6XNN sets VX", func(t *testing.T) {
		m := loadOpcodes(t, 0x6A2F) // set VA = 0x2F
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if m.V[0xA] != 0x2F {
			t.Errorf("V[A] = 0x%02X, want 0x2F", m.V[0xA])
		}
	})

	t.Run("7XNN adds to VX without touching VF", func(t *testing.T) {
		m := loadOpcodes(t, 0x6005, 0x7003) // V0 = 5, then V0 += 3
		m.V[0xF] = 1                        // sentinel: 7XNN must leave VF unchanged
		mustStep(t, m, 2)
		if m.V[0] != 8 {
			t.Errorf("V0 = %d, want 8", m.V[0])
		}
		if m.V[0xF] != 1 {
			t.Errorf("VF = %d, want 1 unchanged (7XNN must not modify VF)", m.V[0xF])
		}
	})

	t.Run("7XNN wraps at 255 without setting carry", func(t *testing.T) {
		m := loadOpcodes(t, 0x60FF, 0x7002) // V0 = 0xFF, then V0 += 2
		m.V[0xF] = 1                        // sentinel: even on overflow VF must not change
		mustStep(t, m, 2)
		if m.V[0] != 0x01 {
			t.Errorf("V0 = 0x%02X, want 0x01 (wraparound)", m.V[0])
		}
		if m.V[0xF] != 1 {
			t.Errorf("VF = %d, want 1 unchanged (7XNN must not set carry on overflow)", m.V[0xF])
		}
	})

	t.Run("ANNN sets I", func(t *testing.T) {
		m := loadOpcodes(t, 0xA123)
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if m.I != 0x123 {
			t.Errorf("I = 0x%03X, want 0x123", m.I)
		}
	})

	t.Run("00E0 clears the display", func(t *testing.T) {
		m := loadOpcodes(t, 0x00E0)
		m.Display[10][10] = true // dirty a pixel first
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		for x := 0; x < DisplayWidth; x++ {
			for y := 0; y < DisplayHeight; y++ {
				if m.Display[x][y] {
					t.Fatalf("Display[%d][%d] still set after 00E0", x, y)
				}
			}
		}
	})

	t.Run("returns error on unimplemented opcode", func(t *testing.T) {
		m := loadOpcodes(t, 0x5000) // 5XY0 (skip-if-equal) not implemented yet
		if err := m.Step(); err == nil {
			t.Fatal("Step returned nil error for unimplemented opcode, want an error")
		}
	})
}

// TestDXYN covers the drawing opcode's semantics: rendering, XOR erase, the VF
// collision flag, coordinate wrapping, and edge clipping.
func TestDXYN(t *testing.T) {
	t.Run("draws a sprite from memory at I", func(t *testing.T) {
		// Draw the built-in "0" glyph (5 bytes at FontStart) at (0,0).
		//
		//   byte   bits       pixels
		//   0xF0   11110000    ####
		//   0x90   10010000    #..#
		//   0x90   10010000    #..#
		//   0x90   10010000    #..#
		//   0xF0   11110000    ####
		//
		//   before:        (0,0) ............
		//                        ............
		//                        ............
		//                        ............
		//                        ............
		//   after draw:    (0,0) ####........   VF=0 (screen was clear, no collision)
		//                        #..#........
		//                        #..#........
		//                        #..#........
		//                        ####........
		m := loadOpcodes(t, 0xD015) // draw at (V0, V1), height 5
		m.V[0], m.V[1] = 0, 0
		m.I = FontStart
		m.V[0xF] = 1 // sentinel: a no-collision draw must reset VF to 0
		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		// "0" glyph top row is 0xF0 => pixels at x=0..3 lit, x=4 dark.
		for x := 0; x < 4; x++ {
			if !m.Display[x][0] {
				t.Errorf("Display[%d][0] not set, expected top row of '0' glyph", x)
			}
		}
		if m.Display[4][0] {
			t.Errorf("Display[4][0] set, expected dark")
		}
		if m.V[0xF] != 0 {
			t.Errorf("VF = %d, want 0 (no collision must reset VF)", m.V[0xF])
		}
	})

	t.Run("XOR erases and sets the collision flag", func(t *testing.T) {
		// A one-pixel sprite (0x80 = 10000000, only the top bit set) drawn twice
		// at (0,0). XOR means the second draw erases the first:
		//
		//   sprite:  #.......
		//
		//   before:        (0,0) ....
		//   after 1st draw:      #...   VF=0 (screen was clear, no collision)
		//   after 2nd draw:      ....   VF=1 (a lit pixel was turned off = collision)
		m := loadOpcodes(t, 0xD011, 0xD011)
		m.V[0], m.V[1] = 0, 0
		m.I = 0x300
		m.Memory[0x300] = 0x80
		m.V[0xF] = 1 // sentinel: the first (no-collision) draw must reset VF to 0

		if err := m.Step(); err != nil {
			t.Fatalf("first Step: %v", err)
		}
		if !m.Display[0][0] {
			t.Fatal("Display[0][0] not set after first draw")
		}
		if m.V[0xF] != 0 {
			t.Errorf("VF = %d after first draw, want 0", m.V[0xF])
		}

		if err := m.Step(); err != nil {
			t.Fatalf("second Step: %v", err)
		}
		if m.Display[0][0] {
			t.Error("Display[0][0] still set after XOR redraw, want cleared")
		}
		if m.V[0xF] != 1 {
			t.Errorf("VF = %d after collision, want 1", m.V[0xF])
		}
	})

	t.Run("wraps the starting coordinate", func(t *testing.T) {
		// A one-pixel sprite (0x80 = 10000000) drawn at (64, 32). Both coordinates
		// are one past the edge, so the starting position wraps modulo the screen
		// size:  X: 64 % 64 = 0,  Y: 32 % 32 = 0
		//
		//   sprite:  #.......
		//
		//   before:      (0,0) ............
		//                      ............
		//   after draw:  (0,0) #...........   VF=0 (pixel lands top-left, not off-screen)
		//                      ............
		m := loadOpcodes(t, 0xD011)
		m.V[0], m.V[1] = DisplayWidth, DisplayHeight
		m.I = 0x300
		m.Memory[0x300] = 0x80

		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if !m.Display[0][0] {
			t.Error("Display[0][0] not set; starting coordinate should wrap")
		}
	})

	t.Run("clips pixels past the right edge", func(t *testing.T) {
		// A two-pixel sprite (0xC0 = 11000000) drawn with its left edge at the
		// last column, x=63. The first pixel lands; the second would fall at
		// x=64, past the right edge, so it is clipped (dropped), NOT wrapped to
		// column 0:
		//
		//   sprite:  ##......
		//
		//   col:          0  1 ...  62  63 | 64 (off-screen)
		//   before:       .  . ...   .   . |  .
		//   after draw:   .  . ...   .   # |  x   VF=0; 2nd px clipped, not wrapped to col 0
		m := loadOpcodes(t, 0xD011)
		m.V[0], m.V[1] = DisplayWidth-1, 0
		m.I = 0x300
		m.Memory[0x300] = 0xC0

		if err := m.Step(); err != nil {
			t.Fatalf("Step: %v", err)
		}
		if !m.Display[DisplayWidth-1][0] {
			t.Error("Display[63][0] not set, expected the in-bounds pixel")
		}
		if m.Display[0][0] {
			t.Error("Display[0][0] set, clipped pixel must not wrap around")
		}
	})
}

// mustStep runs n cycles, failing the test on the first error.
func mustStep(t *testing.T, m *Machine, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := m.Step(); err != nil {
			t.Fatalf("Step %d: %v", i, err)
		}
	}
}
