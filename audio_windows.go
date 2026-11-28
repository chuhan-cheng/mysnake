package main

import (
	_ "embed"
	"runtime"
	"unsafe"
)

//go:embed pickup.wav
var pickupWAV []byte

var (
	waveOutOpen      = winmm.NewProc("waveOutOpen")
	waveOutPrepare   = winmm.NewProc("waveOutPrepareHeader")
	waveOutWrite     = winmm.NewProc("waveOutWrite")
	waveOutReset     = winmm.NewProc("waveOutReset")
	waveOutUnprepare = winmm.NewProc("waveOutUnprepareHeader")
	waveOutClose     = winmm.NewProc("waveOutClose")
	pickup           = pickupPlayer{}
	audioIdle        = pickupPlayer{}
)

type waveHeader struct {
	data             *byte
	length, recorded uint32
	user             uintptr
	flags, loops     uint32
	next, reserved   uintptr
}

type pickupPlayer struct {
	device uintptr
	header waveHeader
	pin    runtime.Pinner
}

func (p *pickupPlayer) open(samples []byte) bool {
	// pickup.wav has a 16-byte PCM format chunk at offset 20 and sample data
	// starting at offset 44. WAVEFORMATEX adds two zero bytes for cbSize.
	var format [18]byte
	copy(format[:], pickupWAV[20:36])
	var device uintptr
	const defaultWaveDevice = 0xffffffff
	if code, _, _ := waveOutOpen.Call(uintptr(unsafe.Pointer(&device)), defaultWaveDevice, uintptr(unsafe.Pointer(&format[0])), 0, 0, 0); code != 0 {
		return false
	}

	p.header = waveHeader{data: &samples[0], length: uint32(len(samples))}
	// WinMM keeps both addresses until the buffer has been unprepared.
	p.pin.Pin(&p.header)
	p.pin.Pin(p.header.data)
	if code, _, _ := waveOutPrepare.Call(device, uintptr(unsafe.Pointer(&p.header)), unsafe.Sizeof(p.header)); code != 0 {
		waveOutClose.Call(device)
		p.pin.Unpin()
		return false
	}
	p.device = device
	return true
}

func startAudio() {
	if pickup.device == 0 {
		pickup.open(pickupWAV[44:])
	}
	if pickup.device == 0 || audioIdle.device != 0 {
		return
	}
	// Keep the shared endpoint streaming between short effects.
	silence := make([]byte, 44100*2/10)
	if audioIdle.open(silence) {
		const loopBuffer = 0x0000000C // WHDR_BEGINLOOP | WHDR_ENDLOOP
		audioIdle.header.flags |= loopBuffer
		audioIdle.header.loops = ^uint32(0)
		if !audioIdle.play() {
			audioIdle.close()
		}
	}
}

func stopAudio() {
	audioIdle.close()
	pickup.close()
}

func (p *pickupPlayer) play() bool {
	if p.device == 0 {
		return false
	}
	code, _, _ := waveOutWrite.Call(p.device, uintptr(unsafe.Pointer(&p.header)), unsafe.Sizeof(p.header))
	const waveStillPlaying = 33
	if code == waveStillPlaying {
		p.stop()
		code, _, _ = waveOutWrite.Call(p.device, uintptr(unsafe.Pointer(&p.header)), unsafe.Sizeof(p.header))
	}
	return code == 0
}

func (p *pickupPlayer) stop() {
	if p.device != 0 {
		waveOutReset.Call(p.device)
	}
}

func (p *pickupPlayer) close() {
	if p.device == 0 {
		return
	}
	p.stop()
	waveOutUnprepare.Call(p.device, uintptr(unsafe.Pointer(&p.header)), unsafe.Sizeof(p.header))
	waveOutClose.Call(p.device)
	p.device = 0
	p.pin.Unpin()
}

func playPickup() {
	if !muted && !pickup.play() {
		playCue("SystemAsterisk")
	}
}
