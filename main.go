// Command chip8 loads a CHIP-8 ROM, runs it until the program settles, and
// prints the resulting display to the terminal.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"chip8-emu/chip8"
)

const (
	// maxCycles bounds execution so a ROM that never settles into an infinite
	// self-loop still terminates instead of hanging.
	maxCycles = 1_000_000

	onPixel  = '#'
	offPixel = ' '
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "chip8:", err)
		os.Exit(1)
	}
}

// run is the real entry point, split out from main so it can return errors
// (and be tested) instead of calling os.Exit directly. By default it opens a
// window; with -tui it runs to completion and prints the display to out.
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("chip8", flag.ContinueOnError)
	fs.SetOutput(out)
	tui := fs.Bool("tui", false, "render to the terminal instead of opening a window")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: chip8 [-tui] <rom-file>")
	}
	path := fs.Arg(0)

	rom, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading ROM: %w", err)
	}

	m := chip8.New()
	if err := m.Load(rom); err != nil {
		return fmt.Errorf("loading ROM: %w", err)
	}

	if *tui {
		return execute(m, out)
	}
	return runWindow(m)
}

// execute steps the machine until it settles into a single-instruction infinite
// loop (the idiom a ROM uses to signal "done" — e.g. the IBM logo ends by
// jumping to itself), an opcode error occurs, or maxCycles is reached. It then
// renders the final display to out.
func execute(m *chip8.Machine, out io.Writer) error {
	for cycle := 0; cycle < maxCycles; cycle++ {
		pc := m.PC
		if err := m.Step(); err != nil {
			return err
		}
		if m.PC == pc { // a jump to itself: the program has settled
			break
		}
	}
	_, err := io.WriteString(out, render(m))
	return err
}

// render returns the 64x32 display as text, one row per line, using onPixel for
// lit pixels and offPixel for dark ones.
func render(m *chip8.Machine) string {
	var b strings.Builder
	b.Grow((chip8.DisplayWidth + 1) * chip8.DisplayHeight)
	for y := 0; y < chip8.DisplayHeight; y++ {
		for x := 0; x < chip8.DisplayWidth; x++ {
			if m.Display[x][y] {
				b.WriteRune(onPixel)
			} else {
				b.WriteRune(offPixel)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
