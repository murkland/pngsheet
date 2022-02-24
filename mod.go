package pngsheet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"os"

	"github.com/yumland/pngchunks"
)

type Info struct {
	SuggestedPalettes map[string]color.Palette
	Frames            []Frame
	Animations        []Animation
}

type Frame struct {
	Left, Top, Right, Bottom, OriginX, OriginY int
}

type Animation struct {
	Frames    []int
	IsLooping bool
}

type action uint8

const (
	actionNext action = 0
	actionLoop action = 1
	actionStop action = 2
)

func LoadInfo(f io.Reader) (Info, error) {
	var info Info
	info.SuggestedPalettes = make(map[string]color.Palette)

	pngr, err := pngchunks.NewReader(f)
	if err != nil {
		return info, err
	}

	for {
		chunk, err := pngr.NextChunk()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
		}

		switch chunk.Type() {
		case "sPLT":
			buf, err := io.ReadAll(chunk)
			if err != nil {
				return info, err
			}
			sepIdx := bytes.IndexByte(buf, '\x00')
			plt := buf[sepIdx+2:]
			var palette color.Palette
			for {
				c := plt[:4]
				palette = append(palette, color.RGBA{c[0], c[1], c[2], c[3]})

				plt = plt[6:]
				if len(plt) == 0 {
					break
				}
			}
			info.SuggestedPalettes[string(buf[:sepIdx])] = palette
		case "zTXt":
			buf, err := io.ReadAll(chunk)
			if err != nil {
				return info, err
			}
			var animation Animation
			ctrlr := bytes.NewReader(buf[bytes.IndexByte(buf, '\x00')+2:])
			for i := 0; ; i++ {
				var rawFrame struct {
					Left, Top, Right, Bottom, OriginX, OriginY int16

					Delay  uint8
					Action action
				}
				if err := binary.Read(ctrlr, binary.LittleEndian, &rawFrame); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return info, err
				}

				frame := Frame{int(rawFrame.Left), int(rawFrame.Top), int(rawFrame.Right), int(rawFrame.Bottom), int(rawFrame.OriginX), int(rawFrame.OriginY)}
				info.Frames = append(info.Frames, frame)

				for j := 0; j < int(rawFrame.Delay); j++ {
					animation.Frames = append(animation.Frames, i)
				}

				if rawFrame.Action != actionNext {
					animation.IsLooping = rawFrame.Action == actionLoop
					info.Animations = append(info.Animations, animation)
					animation = Animation{}
				}
			}
		default:
			if _, err := io.Copy(io.Discard, chunk); err != nil {
				return info, err
			}
		}

		if err := chunk.Close(); err != nil {
			return info, err
		}
	}

	return info, nil
}

var ErrInvalidFormat = errors.New("invalid format")

func Load(f io.ReadSeeker) (image.Image, Info, error) {
	info, err := LoadInfo(f)
	if err != nil {
		return nil, Info{}, err
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, Info{}, err
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, Info{}, err
	}

	return img, info, err
}
