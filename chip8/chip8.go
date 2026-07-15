// Package chip8 implements the state and behavior of the CHIP-8 virtual machine.
package chip8

import "fmt"

const (
	// MemorySize is the total addressable memory. CHIP-8 has 4KB.
	MemorySize = 4096

	// ProgramStart is where ROMs are loaded and where the PC begins.
	// Addresses below this were historically occupied by the interpreter itself.
	ProgramStart = 0x200

	// FontStart is the conventional location of the built-in hex font sprites.
	FontStart = 0x50

	// DisplayWidth and DisplayHeight are the dimensions of the monochrome display.
	DisplayWidth  = 64
	DisplayHeight = 32
)

// Machine holds the complete state of a CHIP-8 machine. A zero-value Machine is
// not ready to run; use New to obtain an initialized machine.
type Machine struct {
	// Memory is the 4KB of addressable RAM. The font lives at FontStart and
	// the loaded program begins at ProgramStart.
	Memory [MemorySize]byte

	// V is the set of 16 general-purpose 8-bit registers, V0..VF
	// VF doubles as a flag register (carry, borrow, collision) and should not
	// be relied upon by programs as general storage.
	V [16]byte

	// I is the index register, used to point at memory addresses (mainly for
	// sprite drawing and load/store). Only the low 12 bits are meaningful.
	I uint16

	// PC is the program counter: the address of the next opcode to fetch.
	PC uint16

	// Stack stores return addresses for CALL (2NNN) / RET (00EE).
	// Original CHIP-8 allowed a limited nesting depth.
	Stack [16]uint16

	// SP is the stack pointer: the index of the next free Stack slot.
	SP byte

	// DelayTimer counts down at 60Hz while non-zero. Programs read it for timing.
	DelayTimer byte

	// SoundTimer counts down at 60Hz while non-zero; a tone plays while it is > 0.
	SoundTimer byte

	// Display is the 64x32 monochrome framebuffer. Each cell is on or off.
	// Indexed as Display[x][y].
	Display [DisplayWidth][DisplayHeight]bool

	// Keys holds the pressed state of the 16-key hex keypad (0x0..0xF).
	Keys [16]bool
}

// fontSet is the standard set of 16 hex-digit sprites (0-F), 5 bytes each.
// Each digit is 4 pixels wide and 5 tall; the high nibble of each byte encodes
// the lit pixels for that row. Programs locate these via the FX29 opcode.
var fontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

// New returns an initialized Machine: the font is loaded into memory and the PC
// is positioned at ProgramStart, ready for a ROM to be loaded.
func New() *Machine {
	m := &Machine{
		PC: ProgramStart,
	}
	copy(m.Memory[FontStart:], fontSet[:])
	return m
}

// MaxROMSize is the largest ROM that fits in memory: everything from
// ProgramStart to the top of RAM.
const MaxROMSize = MemorySize - ProgramStart

// Load copies a ROM image into program memory starting at ProgramStart, where
// the program counter is already positioned. It returns an error if the ROM is
// too large to fit in the available memory.
func (m *Machine) Load(rom []byte) error {
	if len(rom) > MaxROMSize {
		return fmt.Errorf("rom too large: %d bytes, max %d", len(rom), MaxROMSize)
	}
	copy(m.Memory[ProgramStart:], rom)
	return nil
}

// Step fetches, decodes, and executes a single instruction, advancing the
// machine by one cycle. It returns an error if it encounters an opcode that is
// not yet implemented, naming the opcode and the address it was fetched from.
//
// Only the handful of opcodes needed to run the IBM logo ROM are implemented so
// far: 00E0, 1NNN, 6XNN, 7XNN, ANNN, and DXYN.
func (m *Machine) Step() error {
	// Fetch: opcodes are two bytes, stored big-endian.
	opcode := uint16(m.Memory[m.PC])<<8 | uint16(m.Memory[m.PC+1])
	// Advance past the opcode now, so a jump/call simply overwrites PC.
	m.PC += 2

	// Decode the operand fields (see the standard CHIP-8 opcode table).
	var (
		x   = (opcode >> 8) & 0x0F  // 2nd nibble: a register index
		y   = (opcode >> 4) & 0x0F  // 3rd nibble: a register index
		n   = opcode & 0x000F       // 4th nibble: a 4-bit immediate
		nn  = byte(opcode & 0x00FF) // low byte: an 8-bit immediate
		nnn = opcode & 0x0FFF       // low 12 bits: an address
	)

	// Execute: switch on the high nibble, with an inner switch for families
	// that share it.
	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode {
		case 0x00E0: // 00E0: clear the display
			m.Display = [DisplayWidth][DisplayHeight]bool{}
		default:
			return m.unknownOpcode(opcode)
		}
	case 0x1000: // 1NNN: jump to NNN
		m.PC = nnn
	case 0x6000: // 6XNN: set VX = NN
		m.V[x] = nn
	case 0x7000: // 7XNN: add NN to VX (does not affect the carry flag)
		m.V[x] += nn
	case 0xA000: // ANNN: set I = NNN
		m.I = nnn
	case 0xD000: // DXYN: draw an N-byte sprite at (VX, VY)
		m.drawSprite(m.V[x], m.V[y], n)
	default:
		return m.unknownOpcode(opcode)
	}
	return nil
}

// unknownOpcode builds the error returned when Step meets an opcode it does not
// implement. PC has already advanced past the opcode, so the source address is
// PC-2.
func (m *Machine) unknownOpcode(opcode uint16) error {
	return fmt.Errorf("unknown opcode %04X at 0x%03X", opcode, m.PC-2)
}

// drawSprite implements DXYN with original CHIP-8 semantics: the starting
// coordinate wraps around the screen, the N-byte sprite at I is XORed onto the
// display one pixel at a time, pixels that fall past the right or bottom edge
// are clipped (not wrapped), and VF is set to 1 if any lit pixel is turned off
// (a collision) or 0 otherwise.
func (m *Machine) drawSprite(vx, vy byte, height uint16) {
	startX := int(vx) % DisplayWidth
	startY := int(vy) % DisplayHeight
	m.V[0xF] = 0

	for row := 0; row < int(height); row++ {
		y := startY + row
		if y >= DisplayHeight { // clip past the bottom edge
			break
		}
		spriteByte := m.Memory[m.I+uint16(row)]
		for col := 0; col < 8; col++ {
			x := startX + col
			if x >= DisplayWidth { // clip past the right edge
				break
			}
			// Sprite bits are read most-significant first (left to right).
			if spriteByte&(0x80>>col) == 0 {
				continue
			}
			if m.Display[x][y] {
				m.V[0xF] = 1 // a lit pixel is being turned off: collision
			}
			m.Display[x][y] = !m.Display[x][y]
		}
	}
}
