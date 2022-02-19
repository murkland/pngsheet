package pngsheet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"os"

	"github.com/undernet/pngchunks"
)

type Info struct {
	Palette    color.Palette
	Frames     []Frame
	Animations []Animation
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
			plt := buf[bytes.IndexByte(buf, '\x00')+2:]
			for {
				c := plt[:4]
				plt = plt[6:]
				if len(plt) == 0 {
					break
				}
				info.Palette = append(info.Palette, color.RGBA{c[0], c[1], c[2], c[3]})
			}
		case "zTXt":
			buf, err := io.ReadAll(chunk)
			if err != nil {
				return info, err
			}
			var animation Animation
			ctrlr := bytes.NewReader(buf[bytes.IndexByte(buf, '\x00')+2:])
			for i := 0; ; i++ {
				var left, top, right, bottom, originX, originY int16
				if err := binary.Read(ctrlr, binary.LittleEndian, &left); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return info, err
				}

				if err := binary.Read(ctrlr, binary.LittleEndian, &top); err != nil {
					return info, err
				}

				if err := binary.Read(ctrlr, binary.LittleEndian, &right); err != nil {
					return info, err
				}

				if err := binary.Read(ctrlr, binary.LittleEndian, &bottom); err != nil {
					return info, err
				}

				if err := binary.Read(ctrlr, binary.LittleEndian, &originX); err != nil {
					return info, err
				}

				if err := binary.Read(ctrlr, binary.LittleEndian, &originY); err != nil {
					return info, err
				}

				frame := Frame{int(left), int(top), int(right), int(bottom), int(originX), int(originY)}
				info.Frames = append(info.Frames, frame)

				var delay uint8
				if err := binary.Read(ctrlr, binary.LittleEndian, &delay); err != nil {
					return info, err
				}

				var a action
				if err := binary.Read(ctrlr, binary.LittleEndian, &a); err != nil {
					return info, err
				}

				for j := 0; j < int(delay); j++ {
					animation.Frames = append(animation.Frames, i)
				}

				if a != actionNext {
					animation.IsLooping = a == actionLoop
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

func Load(f io.ReadSeeker) (*image.Paletted, Info, error) {
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

	pimg, ok := img.(*image.Paletted)
	if !ok {
		return nil, Info{}, ErrInvalidFormat
	}

	return pimg, info, err
}
