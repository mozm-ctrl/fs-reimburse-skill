package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func decodeDataURL(t *testing.T, u string) image.Image {
	t.Helper()
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(u, prefix) {
		t.Fatalf("not a jpeg data URL: %.40s", u)
	}
	b, err := base64.StdEncoding.DecodeString(u[len(prefix):])
	if err != nil {
		t.Fatal(err)
	}
	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil || format != "jpeg" {
		t.Fatalf("decode: %v %s", err, format)
	}
	return img
}

func TestResizeTo800(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1600, 1200)))
	u, err := ToDataURL(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeDataURL(t, u).Bounds(); b.Dx() != 800 || b.Dy() != 600 {
		t.Fatalf("got %v, want 800x600", b)
	}
}

func TestSmallImageNotUpscaled(t *testing.T) {
	var buf bytes.Buffer
	jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 300, 500)), nil)
	u, err := ToDataURL(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeDataURL(t, u).Bounds(); b.Dx() != 300 || b.Dy() != 500 {
		t.Fatalf("got %v, want 300x500", b)
	}
}

// withOrientation inserts an APP1 EXIF segment carrying only the Orientation tag.
func withOrientation(jpg []byte, o uint16) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01")
	entry := make([]byte, 12)
	binary.BigEndian.PutUint16(entry[0:], 0x0112)
	binary.BigEndian.PutUint16(entry[2:], 3)
	binary.BigEndian.PutUint32(entry[4:], 1)
	binary.BigEndian.PutUint16(entry[8:], o)
	tiff = append(append(tiff, entry...), 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(seg)+2))
	out := append([]byte{}, jpg[:2]...)
	out = append(append(out, app1...), seg...)
	return append(out, jpg[2:]...)
}

func TestExifRotation(t *testing.T) {
	// 400x200 landscape, left half red. Orientation 6 = rotate 90° clockwise to display.
	src := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if x < 200 {
				c = color.RGBA{255, 0, 0, 255}
			}
			src.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, src, &jpeg.Options{Quality: 95})
	raw := withOrientation(buf.Bytes(), 6)
	if o := jpegOrientation(raw); o != 6 {
		t.Fatalf("orientation %d, want 6", o)
	}
	u, err := ToDataURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	img := decodeDataURL(t, u)
	if b := img.Bounds(); b.Dx() != 200 || b.Dy() != 400 {
		t.Fatalf("got %v, want 200x400", b)
	}
	// After a clockwise turn the red (former left) half is on top.
	if r, _, bl, _ := img.At(100, 50).RGBA(); r < bl {
		t.Fatalf("top should be red, got r=%d b=%d", r>>8, bl>>8)
	}
}

func TestRejectsNonImage(t *testing.T) {
	if _, err := ToDataURL([]byte("not an image")); err == nil {
		t.Fatal("expected error")
	}
}
