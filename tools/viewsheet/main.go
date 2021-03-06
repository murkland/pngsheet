package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/murkland/pngsheet"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func main() {
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("%s", err)
	}

	img, info, err := pngsheet.Load(f)
	if err != nil {
		log.Fatalf("%s", err)
	}

	var palette color.Palette
	if pimg, ok := img.(*image.Paletted); ok {
		palette = append(pimg.Palette, info.SuggestedPalettes["extra"]...)
	}

	altPalette := info.SuggestedPalettes["alt"]

	tt, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatalf("%s", err)
	}

	const dpi = 72
	fontFace, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("%s", err)
	}

	markerImg := ebiten.NewImage(5, 5)
	for x := 0; x < 5; x++ {
		markerImg.Set(x, 2, color.RGBA{0, 255, 0, 255})
	}
	for y := 0; y < 5; y++ {
		markerImg.Set(2, y, color.RGBA{0, 255, 0, 255})
	}

	log.Printf("palette size: %d, alt palette size: %d", len(palette), len(altPalette))
	ebiten.RunGame(&game{fontFace: fontFace, origImg: img, info: info, palette: palette, altPalette: altPalette, markerImg: markerImg})
}

func frame(anim *pngsheet.Animation, t int) *pngsheet.Frame {
	if len(anim.Frames) == 0 {
		return nil
	}

	if t >= len(anim.Frames) && anim.IsLooping {
		t %= len(anim.Frames)
	}

	return anim.Frames[t]
}

type game struct {
	fontFace font.Face

	paused     bool
	origImg    image.Image
	info       *pngsheet.Info
	palette    color.Palette
	altPalette color.Palette

	paletteIdx int
	elapsed    int
	frameIdx   int
	animIdx    int
	img        *ebiten.Image

	markerImg *ebiten.Image
}

func (g *game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return 288, 256
}

func (g *game) Draw(screen *ebiten.Image) {
	anim := g.info.Animations[g.animIdx]
	frame := frame(anim, g.elapsed)

	screen.Fill(color.RGBA{0xff, 0x00, 0xff, 0xff})

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(288/2-frame.Origin.X), float64(256/2-frame.Origin.Y))
	screen.DrawImage(g.img.SubImage(frame.Rect).(*ebiten.Image), opts)
	palInfo := "not paletted"
	if g.palette != nil {
		palInfo = fmt.Sprintf("palette: %03d/%03d", g.paletteIdx+1, len(g.palette)/16)
	}
	opts2 := &ebiten.DrawImageOptions{}
	opts2.GeoM.Translate(float64(288/2-2), float64(256/2-2))
	screen.DrawImage(g.markerImg, opts2)
	text.Draw(screen, fmt.Sprintf("%s\nanim: %d\nelapsed: %d\nframe: %d", palInfo, g.animIdx, g.elapsed, frame.Index), g.fontFace, 4, 12+4, color.RGBA{0x00, 0xff, 0x00, 0xff})
}

func (g *game) shiftPalette(i int) {
	pimg := g.origImg.(*image.Paletted)
	g.paletteIdx = i
	pimg.Palette = g.palette[g.paletteIdx*16:]
	for len(pimg.Palette) < 256 {
		pimg.Palette = append(pimg.Palette, color.RGBA{})
	}
	g.img = ebiten.NewImageFromImage(pimg)
}

func (g *game) swapPalette() {
	g.palette, g.altPalette = g.altPalette, g.palette
	g.shiftPalette(g.paletteIdx)
}

func (g *game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}

	if g.img == nil {
		g.img = ebiten.NewImageFromImage(g.origImg)
		g.elapsed = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if g.palette != nil {
			i := (g.paletteIdx + 1) % (len(g.palette) / 16)
			if i < 0 {
				i += (len(g.palette) / 16)
			}
			g.shiftPalette(i)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if g.palette != nil {
			i := (g.paletteIdx - 1) % (len(g.palette) / 16)
			if i < 0 {
				i += (len(g.palette) / 16)
			}
			g.shiftPalette(i)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.animIdx = (g.animIdx + 1) % len(g.info.Animations)
		g.elapsed = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.animIdx = (g.animIdx - 1) % len(g.info.Animations)
		if g.animIdx < 0 {
			g.animIdx += len(g.info.Animations)
		}
		g.elapsed = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.elapsed = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if g.palette != nil {
			g.swapPalette()
		}
	}

	anim := g.info.Animations[g.animIdx]
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
		g.paused = true
		if anim.IsLooping || g.elapsed < len(anim.Frames)-1 {
			g.elapsed++
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		g.paused = true
		if g.elapsed > 0 {
			g.elapsed--
		}
	}

	if !g.paused {
		if anim.IsLooping || g.elapsed < len(anim.Frames)-1 {
			g.elapsed++
		}
	}

	return nil
}
