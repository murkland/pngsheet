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
	"github.com/yumland/pngsheet"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func main() {
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("%s", err)
	}

	pimg, info, err := pngsheet.Load(f)
	if err != nil {
		log.Fatalf("%s", err)
	}

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

	ebiten.RunGame(&game{fontFace: fontFace, origImg: pimg, info: info})
}

func frame(anim pngsheet.Animation, t int) int {
	if len(anim.Frames) == 0 {
		return 0
	}

	if t >= len(anim.Frames) {
		if anim.IsLooping {
			t %= len(anim.Frames)
		} else {
			t = len(anim.Frames) - 1
		}
	}

	return anim.Frames[t]
}

type game struct {
	fontFace font.Face

	origImg *image.Paletted
	info    pngsheet.Info

	paletteIdx int
	elapsed    int
	frameIdx   int
	animIdx    int
	img        *ebiten.Image
}

func (g *game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return 288, 256
}

func (g *game) Draw(screen *ebiten.Image) {
	anim := g.info.Animations[g.animIdx]
	frameIdx := frame(anim, g.elapsed)
	frame := g.info.Frames[frameIdx]

	screen.Fill(color.RGBA{0xff, 0x00, 0xff, 0xff})

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(288/2-frame.OriginX), float64(256/2-frame.OriginY))
	screen.DrawImage(g.img.SubImage(image.Rect(frame.Left, frame.Top, frame.Right, frame.Bottom)).(*ebiten.Image), opts)
	text.Draw(screen, fmt.Sprintf("palette: %d\nanim: %d\nframe: %d", g.paletteIdx, g.animIdx, frameIdx-anim.Frames[0]), g.fontFace, 4, 12+4, color.RGBA{0x00, 0xff, 0x00, 0xff})
}

func (g *game) swapPalette(i int) {
	g.paletteIdx = i
	g.origImg.Palette = g.info.Palette[g.paletteIdx*16:]
	for len(g.origImg.Palette) < 256 {
		g.origImg.Palette = append(g.origImg.Palette, color.RGBA{})
	}
	g.img = ebiten.NewImageFromImage(g.origImg)
}

func (g *game) Update() error {
	g.elapsed++

	if g.img == nil {
		g.img = ebiten.NewImageFromImage(g.origImg)
		g.elapsed = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.swapPalette((g.paletteIdx + 1) % (len(g.info.Palette) / 16))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		i := (g.paletteIdx - 1) % (len(g.info.Palette) / 16)
		if i < 0 {
			i += (len(g.info.Palette) / 16)
		}
		g.swapPalette(i)
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

	return nil
}
