// Command chip8 loads a CHIP-8 ROM into a fresh machine. Running the loaded
// program is added in a later step; for now it reads and loads the ROM,
// reporting what it did.
package main

import (
	"fmt"
	"os"

	"chip8-emu/chip8"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "chip8:", err)
		os.Exit(1)
	}
}

// run is the real entry point, split out from main so it can return errors
// (and be tested) instead of calling os.Exit directly.
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: chip8 <rom-file>")
	}
	path := args[0]

	rom, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading ROM: %w", err)
	}

	m := chip8.New()
	if err := m.Load(rom); err != nil {
		return fmt.Errorf("loading ROM: %w", err)
	}

	fmt.Printf("loaded %q: %d bytes into memory at 0x%03X\n", path, len(rom), chip8.ProgramStart)
	return nil
}
