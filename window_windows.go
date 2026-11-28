package main

import (
	"log"
	"runtime"
	"syscall"
	"time"
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
	setTimer      = user.NewProc("SetTimer")
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
var movementInterval int

func updateMovementTimer(window uintptr) error {
	interval := moveInterval(current.score)
	if interval == movementInterval {
		return nil
	}
	if timer, _, err := setTimer.Call(window, 1, uintptr(interval), 0); timer == 0 {
		return err
	}
	movementInterval = interval
	return nil
}

func main() {
	runtime.LockOSThread()
	current.reset()
	if handle := openHistory(); handle != 0 {
		defer syscall.CloseHandle(handle)
	}
	if !muted {
		startAudio()
	}
	defer stopAudio()
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
		0x10CF0000, 0x80000000, 0x80000000, 810, 780, 0, 0, instance, 0,
	)
	if window == 0 {
		log.Fatal(err)
	}
	if err := updateMovementTimer(window); err != nil {
		log.Fatal(err)
	}
	setTimer.Call(window, 2, 25, 0)
	setTimer.Call(window, 3, 65, 0)
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
		finishRound(time.Now(), abandoned)
		user.NewProc("PostQuitMessage").Call(0)
	case 0x14: // WM_ERASEBKGND
		return 1
	case 0x0F: // WM_PAINT
		render(window)
	case 5: // WM_SIZE
		bounds := rect{right: int32(uint16(flags)), bottom: int32(uint16(flags >> 16))}
		historyPage = min(historyPage, pageCount(bounds)-1)
	case 0x113: // WM_TIMER
		now := time.Now()
		if key == 1 && current.started && !current.paused && !current.ended {
			advance(now)
			_ = updateMovementTimer(window)
			invalidate.Call(window, 0, 0)
		}
		if key == 2 && (now.Sub(foodAt) < foodEffect+25*time.Millisecond || now.Sub(impactAt) < impactEffect+25*time.Millisecond) {
			invalidate.Call(window, 0, 0)
		}
		if key == 3 && !showHistory && !current.paused && !current.ended {
			invalidate.Call(window, 0, 0)
		}
	case 0x202: // WM_LBUTTONUP
		var bounds rect
		getClientRect.Call(window, uintptr(unsafe.Pointer(&bounds)))
		x, y := int32(int16(flags)), int32(int16(flags>>16))
		for _, button := range controls(bounds) {
			if button.enabled && x >= button.area.left && x < button.area.right &&
				y >= button.area.top && y < button.area.bottom {
				handleKey(window, button.key, time.Now())
				_ = updateMovementTimer(window)
				break
			}
		}
		invalidate.Call(window, 0, 0)
	case 0x100: // WM_KEYDOWN
		if flags&(1<<30) == 0 {
			handleKey(window, key, time.Now())
			_ = updateMovementTimer(window)
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

func fill(dc uintptr, area rect, color uintptr) {
	brush, _, _ := createBrush.Call(color)
	fillRect.Call(dc, uintptr(unsafe.Pointer(&area)), brush)
	deleteObject.Call(brush)
}

func label(dc uintptr, area rect, value string, height int32, color uintptr) {
	font, _, _ := gdi.NewProc("CreateFontW").Call(uintptr(-height), 0, 0, 0, 600, 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("Microsoft JhengHei UI"))))
	defer deleteObject.Call(font)
	old, _, _ := selectObject.Call(dc, font)
	defer selectObject.Call(dc, old)
	gdi.NewProc("SetBkMode").Call(dc, 1)
	gdi.NewProc("SetTextColor").Call(dc, color)
	drawText.Call(dc, uintptr(unsafe.Pointer(wide(value))), ^uintptr(0), uintptr(unsafe.Pointer(&area)), 0x25)
}
