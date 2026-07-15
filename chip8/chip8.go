// Package chip8 implements the state and behavior of the CHIP-8 virtual machine.
package chip8

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
