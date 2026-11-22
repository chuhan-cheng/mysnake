package main

import (
	"log"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user   = syscall.NewLazyDLL("user32.dll")
	gdi    = syscall.NewLazyDLL("gdi32.dll")
	kernel = syscall.NewLazyDLL("kernel32.dll")

	beginPaint    = user.NewProc("BeginPaint")
	endPaint      = user.NewProc("EndPaint")
	getClientRect = user.NewProc("GetClientRect")
	fillRect      = user.NewProc("FillRect")
	drawText      = user.NewProc("DrawTextW")
	invalidate    = user.NewProc("InvalidateRect")
	createBrush   = gdi.NewProc("CreateSolidBrush")
	deleteObject  = gdi.NewProc("DeleteObject")
	selectObject  = gdi.NewProc("SelectObject")
)

type rect struct{ left, top, right, bottom int32 }
type point struct{ x, y int32 }
type paint struct {
	dc              uintptr
	erase           int32
	area            rect
	restore, update int32
	reserved        [32]byte
}
type message struct {
	window         uintptr
	id             uint32
	wparam, lparam uintptr
	time           uint32
	position       point
	private        uint32
}
type windowClass struct {
	style                              uint32
	proc                               uintptr
	classExtra, windowExtra            int32
	instance, icon, cursor, background uintptr
	menu, name                         *uint16
}

var current game

func main() {
	runtime.LockOSThread()
	current.reset()
	instance, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	cursor, _, _ := user.NewProc("LoadCursorW").Call(0, 32512)
	icon, _, _ := user.NewProc("LoadIconW").Call(instance, 1)
	name := wide("GoSnake")
	class := windowClass{style: 3, proc: syscall.NewCallback(windowProc), instance: instance, icon: icon, cursor: cursor, name: name}
	if result, _, err := user.NewProc("RegisterClassW").Call(uintptr(unsafe.Pointer(&class))); result == 0 {
		log.Fatal(err)
	}
	window, _, err := user.NewProc("CreateWindowExW").Call(
		0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(wide("貪食蛇"))),
		0x10CF0000, 0x80000000, 0x80000000, 670, 650, 0, 0, instance, 0,
	)
	if window == 0 {
		log.Fatal(err)
	}
	if timer, _, err := user.NewProc("SetTimer").Call(window, 1, 115, 0); timer == 0 {
		log.Fatal(err)
	}
	getMessage := user.NewProc("GetMessageW")
	translate := user.NewProc("TranslateMessage")
	dispatch := user.NewProc("DispatchMessageW")
	var msg message
	for {
		result, _, err := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(result) {
		case 0:
			return
		case -1:
			log.Fatal(err)
		}
		translate.Call(uintptr(unsafe.Pointer(&msg)))
		dispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func wide(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func windowProc(window uintptr, msg uint32, key, flags uintptr) uintptr {
	switch msg {
	case 2: // WM_DESTROY
		user.NewProc("PostQuitMessage").Call(0)
	case 0x14: // WM_ERASEBKGND
		return 1
	case 0x0F: // WM_PAINT
		render(window)
	case 0x113: // WM_TIMER
		if current.started && !current.paused && !current.ended {
			current.step()
			invalidate.Call(window, 0, 0)
		}
	case 0x100: // WM_KEYDOWN
		switch key {
		case 0x25, 'A':
			current.turn(cell{-1, 0})
		case 0x26, 'W':
			current.turn(cell{0, -1})
		case 0x27, 'D':
			current.turn(cell{1, 0})
		case 0x28, 'S':
			current.turn(cell{0, 1})
		case ' ':
			if flags&(1<<30) == 0 && current.started && !current.ended {
				current.paused = !current.paused
			}
		case 'R', 0x0D:
			if flags&(1<<30) == 0 {
				current.reset()
			}
		}
		invalidate.Call(window, 0, 0)
	default:
		result, _, _ := user.NewProc("DefWindowProcW").Call(window, uintptr(msg), key, flags)
		return result
	}
	return 0
}

func render(window uintptr) {
	var ps paint
	dc, _, _ := beginPaint.Call(window, uintptr(unsafe.Pointer(&ps)))
	defer func() {
		endPaint.Call(window, uintptr(unsafe.Pointer(&ps)))
	}()
	var bounds rect
	getClientRect.Call(window, uintptr(unsafe.Pointer(&bounds)))
	if bounds.right == 0 || bounds.bottom == 0 {
		return
	}
	buffer, _, _ := gdi.NewProc("CreateCompatibleDC").Call(dc)
	if buffer != 0 {
		defer gdi.NewProc("DeleteDC").Call(buffer)
		bitmap, _, _ := gdi.NewProc("CreateCompatibleBitmap").Call(dc, uintptr(bounds.right), uintptr(bounds.bottom))
		if bitmap != 0 {
			defer deleteObject.Call(bitmap)
			old, _, _ := selectObject.Call(buffer, bitmap)
			defer selectObject.Call(buffer, old)
			drawScene(buffer, bounds)
			gdi.NewProc("BitBlt").Call(dc, 0, 0, uintptr(bounds.right), uintptr(bounds.bottom), buffer, 0, 0, 0xCC0020)
			return
		}
	}
	drawScene(dc, bounds)
}

func drawScene(dc uintptr, bounds rect) {
	fill(dc, bounds, 0x101610)
	size := min((bounds.right-24)/columns, (bounds.bottom-100)/rows)
	if size < 5 {
		label(dc, bounds, "請放大視窗")
		return
	}
	width, height := size*columns, size*rows
	x := (bounds.right - width) / 2
	y := (bounds.bottom - height) / 2
	fill(dc, rect{x - 2, y - 2, x + width + 2, y + height + 2}, 0x456048)
	fill(dc, rect{x, y, x + width, y + height}, 0x1D291E)
	drawCell := func(p cell, color uintptr, inset int32) {
		left, top := x+int32(p.x)*size, y+int32(p.y)*size
		fill(dc, rect{left + inset, top + inset, left + size - inset, top + size - inset}, color)
	}
	if !current.won {
		drawCell(current.food, 0x577AFF, max(1, size/5))
	}
	for i, segment := range current.snake {
		color := uintptr(0x73C448)
		if i == 0 {
			color = 0x99F0A7
		}
		drawCell(segment, color, 1)
	}
	label(dc, rect{0, 0, bounds.right, y}, current.scoreboard())
	label(dc, rect{0, y + height, bounds.right, bounds.bottom}, current.status())
}

func fill(dc uintptr, area rect, color uintptr) {
	brush, _, _ := createBrush.Call(color)
	fillRect.Call(dc, uintptr(unsafe.Pointer(&area)), brush)
	deleteObject.Call(brush)
}

func label(dc uintptr, area rect, value string) {
	font, _, _ := gdi.NewProc("GetStockObject").Call(17)
	old, _, _ := selectObject.Call(dc, font)
	defer selectObject.Call(dc, old)
	gdi.NewProc("SetBkMode").Call(dc, 1)
	gdi.NewProc("SetTextColor").Call(dc, 0xD9E9DE)
	drawText.Call(dc, uintptr(unsafe.Pointer(wide(value))), ^uintptr(0), uintptr(unsafe.Pointer(&area)), 0x25)
}
