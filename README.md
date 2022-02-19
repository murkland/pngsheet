# pngsheet

pngsheet is a PNG file that contains a sprite sheet with animations. It additionally contains two metadata chunks, a supplementary palette in `sPLT` and an animation control section in `zTXt` with the keyword `fsctrl` and "compression scheme" `0xFF`.

For GBA rips, the `sPLT` section is expected to contain groups of 16 colors.

The `fsctrl` section is structured as follows:

```
left:       int16
top:        int16
right:      int16
bottom:     int16
originX:    int16
originY:    int16
delay:      uint8
action:     0x00 "next" | 0x01 "loop" | 0x03 "stop"
```

Each animation is delimited by a non-"next" action.
