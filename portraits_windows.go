package main

import (
	"bytes"
	_ "embed"
	"image/jpeg"
	"math"
	"time"
	"unsafe"
)

//go:embed portraits/chiang.jpg
var chiangPhoto []byte

//go:embed portraits/shen.jpg
var shenPhoto []byte

const portraitSize = 64

var (
	portraitNames  = [...]string{"蔣萬安", "沈伯洋"}
	portraitPixels = [...][portraitSize * portraitSize]uint32{
		portraitBitmap(chiangPhoto, 0.08),
		portraitBitmap(shenPhoto, 0.045),
	}
	stretchDIBits       = gdi.NewProc("StretchDIBits")
	setStretchMode      = gdi.NewProc("SetStretchBltMode")
	createEllipseRegion = gdi.NewProc("CreateEllipticRgn")
	selectClipRegion    = gdi.NewProc("SelectClipRgn")
	saveDC              = gdi.NewProc("SaveDC")
	restoreDC           = gdi.NewProc("RestoreDC")
	// BITMAPINFOHEADER for a top-down, 32-bit RGB bitmap.
	portraitBitmapInfo = [10]uint32{40, portraitSize, ^uint32(portraitSize - 1), 0x00200001}
)

func portraitBitmap(photo []byte, topFraction float64) [portraitSize * portraitSize]uint32 {
	img, err := jpeg.Decode(bytes.NewReader(photo))
	if err != nil {
		panic(err)
	}
	var pixels [portraitSize * portraitSize]uint32
	bounds := img.Bounds()
	side := bounds.Dx() * 85 / 100
	left := bounds.Min.X + (bounds.Dx()-side)/2
	top := bounds.Min.Y + int(float64(bounds.Dy())*topFraction)
	for y := range portraitSize {
		for x := range portraitSize {
			r, g, b, _ := img.At(left+x*side/portraitSize, top+y*side/portraitSize).RGBA()
			pixels[y*portraitSize+x] = (r>>8)<<16 | (g>>8)<<8 | b>>8
		}
	}
	return pixels
}

func drawPortraitFood(dc uintptr, area rect, kind fruit, portrait int, now time.Time) {
	size := area.right - area.left
	offset := int32(math.Round(math.Sin(2*math.Pi*float64(now.UnixMilli()%1000)/1000) * float64(size) / 10))
	face := rect{area.left + 1, area.top + 2 + offset, area.right - 1, area.bottom - 2 + offset}
	rim := [...]uintptr{pink, 0x54CCFF, 0xFFE393}[kind]
	rounded(dc, rect{face.left - 1, face.top - 1, face.right + 1, face.bottom + 1}, rim, size)
	state, _, _ := saveDC.Call(dc)
	if state == 0 {
		return
	}
	defer restoreDC.Call(dc, state)
	region, _, _ := createEllipseRegion.Call(uintptr(face.left), uintptr(face.top), uintptr(face.right), uintptr(face.bottom))
	if region == 0 {
		return
	}
	defer deleteObject.Call(region)
	selectClipRegion.Call(dc, region)
	setStretchMode.Call(dc, 3) // COLORONCOLOR preserves portrait colors when shrinking.
	stretchDIBits.Call(dc, uintptr(face.left), uintptr(face.top), uintptr(face.right-face.left), uintptr(face.bottom-face.top),
		0, 0, portraitSize, portraitSize, uintptr(unsafe.Pointer(&portraitPixels[portrait][0])),
		uintptr(unsafe.Pointer(&portraitBitmapInfo[0])), 0, 0x00CC0020)
}
