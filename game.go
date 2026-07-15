package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"chip8-emu/chip8"
)

const (
	// windowScale is how many window pixels wide/tall each CHIP-8 pixel is.
	windowScale = 12

	// cyclesPerFrame is how many CPU instructions run per 60Hz frame,
	// giving roughly a 600Hz clock.
	cyclesPerFrame = 10
)

// keyMap maps physical keyboard keys to CHIP-8 hex keypad values, using the
// conventional layout:
//
//	Keyboard       CHIP-8
//	1 2 3 4        1 2 3 C
//	Q W E R   -->  4 5 6 D
//	A S D F        7 8 9 E
//	Z X C V        A 0 B F
var keyMap = map[ebiten.Key]byte{
	ebiten.KeyDigit1: 0x1, ebiten.KeyDigit2: 0x2, ebiten.KeyDigit3: 0x3, ebiten.KeyDigit4: 0xC,
	ebiten.KeyQ: 0x4, ebiten.KeyW: 0x5, ebiten.KeyE: 0x6, ebiten.KeyR: 0xD,
	ebiten.KeyA: 0x7, ebiten.KeyS: 0x8, ebiten.KeyD: 0x9, ebiten.KeyF: 0xE,
	ebiten.KeyZ: 0xA, ebiten.KeyX: 0x0, ebiten.KeyC: 0xB, ebiten.KeyV: 0xF,
}

// Game adapts a chip8.Machine to the ebiten game loop.
type Game struct {
	m  *chip8.Machine
	fb *ebiten.Image // 64x32 offscreen framebuffer, scaled up when drawn
}

func newGame(m *chip8.Machine) *Game {
	return &Game{
		m:  m,
		fb: ebiten.NewImage(chip8.DisplayWidth, chip8.DisplayHeight),
	}
}

// runWindow opens a window and runs the machine until the window is closed or
// the machine hits an error (e.g. an unimplemented opcode).
func runWindow(m *chip8.Machine) error {
	ebiten.SetWindowSize(chip8.DisplayWidth*windowScale, chip8.DisplayHeight*windowScale)
	ebiten.SetWindowTitle("CHIP-8")
	return ebiten.RunGame(newGame(m))
}

// Update advances the machine one frame: it samples the keyboard, runs a batch
// of CPU cycles, and ticks the 60Hz timers. ebiten calls it at 60 TPS.
func (g *Game) Update() error {
	for key, value := range keyMap {
		g.m.Keys[value] = ebiten.IsKeyPressed(key)
	}
	for i := 0; i < cyclesPerFrame; i++ {
		if err := g.m.Step(); err != nil {
			return err
		}
	}
	if g.m.DelayTimer > 0 {
		g.m.DelayTimer--
	}
	if g.m.SoundTimer > 0 {
		g.m.SoundTimer--
	}
	return nil
}

// Draw paints the machine's display into the window, scaled up.
func (g *Game) Draw(screen *ebiten.Image) {
	g.fb.WritePixels(framebufferBytes(g.m))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(windowScale, windowScale)
	screen.DrawImage(g.fb, op)
}

// Layout reports the logical screen size; the window is the same size so pixels
// map 1:1 and stay crisp.
func (g *Game) Layout(_, _ int) (int, int) {
	return chip8.DisplayWidth * windowScale, chip8.DisplayHeight * windowScale
}

// framebufferBytes converts the machine's display into a row-major RGBA buffer
// (4 bytes per pixel) sized for the 64x32 framebuffer image: white for lit
// pixels, opaque black for dark ones.
func framebufferBytes(m *chip8.Machine) []byte {
	buf := make([]byte, chip8.DisplayWidth*chip8.DisplayHeight*4)
	for y := 0; y < chip8.DisplayHeight; y++ {
		for x := 0; x < chip8.DisplayWidth; x++ {
			i := (y*chip8.DisplayWidth + x) * 4
			if m.Display[x][y] {
				buf[i], buf[i+1], buf[i+2], buf[i+3] = 0xFF, 0xFF, 0xFF, 0xFF
			} else {
				buf[i+3] = 0xFF // R=G=B=0, A=0xFF -> opaque black
			}
		}
	}
	return buf
}
