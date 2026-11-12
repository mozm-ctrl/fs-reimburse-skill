package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"

	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"

	_ "github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ToDataURL uses the site frontend's encoding: at most 800px wide, JPEG quality 60, as a data URL.
// No external tools (sips, ImageMagick), so it works the same on macOS, Linux and Windows.
func ToDataURL(raw []byte) (string, error) {
	src, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("无法识别的图片格式（支持 JPEG/PNG/HEIC/GIF/WebP/BMP/TIFF）: %w", err)
	}
	if format == "jpeg" {
		src = orient(src, jpegOrientation(raw))
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > 800 {
		w, h = 800, max(1, h*800/b.Dx())
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// White background so transparent PNGs don't turn black in JPEG.
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 60}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

// jpegOrientation reads the EXIF Orientation tag (1-8); phone photos rely on it, and re-encoding
// drops EXIF, so it has to be applied to the pixels. Returns 1 when absent or unreadable.
func jpegOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xFF {
			return 1
		}
		marker := b[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		n := int(binary.BigEndian.Uint16(b[i+2:]))
		seg := b[i+4 : min(len(b), i+2+n)]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return exifOrientation(seg[6:])
		}
		i += 2 + n
	}
	return 1
}

func exifOrientation(t []byte) int {
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	off := int(bo.Uint32(t[4:]))
	if off+2 > len(t) {
		return 1
	}
	count := int(bo.Uint16(t[off:]))
	for k := 0; k < count; k++ {
		e := off + 2 + 12*k
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// orient returns src transformed so it displays upright for the given EXIF orientation.
func orient(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
