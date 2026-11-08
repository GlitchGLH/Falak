package main

import (
	_ "embed"
	"encoding/binary"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	//go:embed res/hover.wav
	hoverWav []byte
	//go:embed res/appear.wav
	appearWav []byte
	//go:embed res/drop.wav
	dropWav []byte
)

const (
	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndMemory    = 0x0004
)

func soundOn() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Falak`, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("SoundOff")
	return !(err == nil && v == 1)
}

func setSoundOn(on bool) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer k.Close()
	if on {
		k.DeleteValue("SoundOff")
	} else {
		k.SetDWordValue("SoundOff", 1)
	}
	soundEnabled = on
}

var soundEnabled = true

// ---- persistent output stream ----

const (
	outRate       = 48000
	bufFrames     = 480 // 10 ms
	bufCount      = 5
	idleClose     = 20 * time.Second
	maxVoices     = 16
	waveMapper    = 0xFFFFFFFF
	callbackEvent = 0x00050000
	whdrDone      = 0x00000001
)

var (
	pWaveOutOpen            = winmm.NewProc("waveOutOpen")
	pWaveOutClose           = winmm.NewProc("waveOutClose")
	pWaveOutPrepareHeader   = winmm.NewProc("waveOutPrepareHeader")
	pWaveOutUnprepareHeader = winmm.NewProc("waveOutUnprepareHeader")
	pWaveOutWrite           = winmm.NewProc("waveOutWrite")
	pWaveOutReset           = winmm.NewProc("waveOutReset")
	sndKernel32             = windows.NewLazySystemDLL("kernel32.dll")
	pSndSetThreadPriority   = sndKernel32.NewProc("SetThreadPriority")
)

type waveFormatEx struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	CbSize         uint16
}

type waveHdr struct {
	Data          uintptr
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

// Globals: fixed addresses the driver can use asynchronously.
// Only one soundLoop goroutine runs at a time.
var (
	sndBufs [bufCount][bufFrames * 2]int16
	sndHdrs [bufCount]waveHdr
)

type pcm []float32 // interleaved stereo, outRate

type voice struct {
	d     pcm
	pos   int
	hover bool
}

var snd struct {
	mu      sync.Mutex
	voices  []voice
	running bool
	lastUse time.Time
}

var (
	decodeOnce                   sync.Once
	hoverPCM, appearPCM, dropPCM pcm
)

func decodeAll() {
	hoverPCM = decodeWav(hoverWav)
	appearPCM = decodeWav(appearWav)
	dropPCM = decodeWav(dropWav)
}

func playHover()  { decodeOnce.Do(decodeAll); playPCM(hoverPCM, hoverWav, true) }
func playAppear() { decodeOnce.Do(decodeAll); playPCM(appearPCM, appearWav, false) }
func playDrop()   { decodeOnce.Do(decodeAll); playPCM(dropPCM, dropWav, false) }

// soundKeepAlive keeps the stream open while the wheel is visible.
func soundKeepAlive() {
	snd.mu.Lock()
	snd.lastUse = time.Now()
	snd.mu.Unlock()
}

func playPCM(d pcm, raw []byte, hover bool) {
	if !soundEnabled {
		return
	}
	if len(d) == 0 {
		playSoundLegacy(raw)
		return
	}
	snd.mu.Lock()
	if hover { // same feel as before: a new tick replaces the previous tick
		keep := snd.voices[:0]
		for _, v := range snd.voices {
			if !v.hover {
				keep = append(keep, v)
			}
		}
		snd.voices = keep
	}
	if len(snd.voices) >= maxVoices {
		snd.voices = snd.voices[1:]
	}
	snd.voices = append(snd.voices, voice{d: d, hover: hover})
	snd.lastUse = time.Now()
	start := !snd.running
	snd.running = true
	snd.mu.Unlock()
	if start {
		go soundLoop()
	}
}

func playSoundLegacy(b []byte) {
	if len(b) == 0 {
		return
	}
	pPlaySoundW.Call(uintptr(unsafe.Pointer(&b[0])), 0, sndMemory|sndAsync|sndNoDefault)
}

func soundLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pSndSetThreadPriority.Call(^uintptr(1), 2) // current thread, HIGHEST
	fails := 0
	for {
		if playOnDevice() {
			fails++
			if fails > 3 {
				logf("sound: device failed repeatedly, using PlaySound fallback")
				snd.mu.Lock()
				snd.voices = nil
				snd.running = false
				snd.mu.Unlock()
				playSoundLegacy(hoverWav)
				return
			}
			time.Sleep(150 * time.Millisecond)
			continue
		}
		snd.mu.Lock()
		if len(snd.voices) == 0 {
			snd.running = false
			snd.mu.Unlock()
			return
		}
		snd.mu.Unlock()
		fails = 0
	}
}

// playOnDevice returns true if the device failed (reopen), false on idle close.
func playOnDevice() bool {
	ev, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		logf("sound: CreateEvent: %v", err)
		return true
	}
	defer windows.CloseHandle(ev)

	wf := waveFormatEx{FormatTag: 1, Channels: 2, SamplesPerSec: outRate,
		AvgBytesPerSec: outRate * 4, BlockAlign: 4, BitsPerSample: 16}
	var h uintptr
	if r, _, _ := pWaveOutOpen.Call(uintptr(unsafe.Pointer(&h)), waveMapper,
		uintptr(unsafe.Pointer(&wf)), uintptr(ev), 0, callbackEvent); r != 0 {
		logf("sound: waveOutOpen failed: %d", r)
		return true
	}
	hsz := unsafe.Sizeof(waveHdr{})
	defer func() {
		pWaveOutReset.Call(h)
		for i := range sndHdrs {
			pWaveOutUnprepareHeader.Call(h, uintptr(unsafe.Pointer(&sndHdrs[i])), hsz)
		}
		pWaveOutClose.Call(h)
	}()

	for i := range sndHdrs {
		sndHdrs[i] = waveHdr{Data: uintptr(unsafe.Pointer(&sndBufs[i][0])), BufferLength: bufFrames * 4}
		if r, _, _ := pWaveOutPrepareHeader.Call(h, uintptr(unsafe.Pointer(&sndHdrs[i])), hsz); r != 0 {
			logf("sound: prepare failed: %d", r)
			return true
		}
	}
	for i := range sndHdrs {
		mixInto(sndBufs[i][:])
		if r, _, _ := pWaveOutWrite.Call(h, uintptr(unsafe.Pointer(&sndHdrs[i])), hsz); r != 0 {
			logf("sound: write failed: %d", r)
			return true
		}
	}
	lastDone := time.Now()
	for {
		windows.WaitForSingleObject(ev, 50)
		done := false
		for i := range sndHdrs {
			if atomic.LoadUint32(&sndHdrs[i].Flags)&whdrDone == 0 {
				continue
			}
			done = true
			mixInto(sndBufs[i][:])
			if r, _, _ := pWaveOutWrite.Call(h, uintptr(unsafe.Pointer(&sndHdrs[i])), hsz); r != 0 {
				logf("sound: write failed (device lost?): %d", r)
				return true
			}
		}
		now := time.Now()
		if done {
			lastDone = now
		} else if now.Sub(lastDone) > time.Second {
			logf("sound: device stalled, reopening")
			return true
		}
		snd.mu.Lock()
		idle := len(snd.voices) == 0 && now.Sub(snd.lastUse) > idleClose
		snd.mu.Unlock()
		if idle {
			return false
		}
	}
}

// mixInto sums active voices at their original level (no gain), clips only.
func mixInto(buf []int16) {
	var acc [bufFrames * 2]float32
	snd.mu.Lock()
	keep := snd.voices[:0]
	for _, v := range snd.voices {
		for i := 0; i < len(acc) && v.pos < len(v.d); i++ {
			acc[i] += v.d[v.pos]
			v.pos++
		}
		if v.pos < len(v.d) {
			keep = append(keep, v)
		}
	}
	snd.voices = keep
	snd.mu.Unlock()
	for i, s := range acc {
		x := s * 32768
		if x > 32767 {
			x = 32767
		} else if x < -32768 {
			x = -32768
		}
		buf[i] = int16(x)
	}
}

// decodeWav: PCM 8/16/24/32, float32, EXTENSIBLE -> stereo float at outRate.
func decodeWav(b []byte) pcm {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil
	}
	var tag, ch, bits uint16
	var rate uint32
	var data []byte
	for p := 12; p+8 <= len(b); {
		id := string(b[p : p+4])
		n := int(binary.LittleEndian.Uint32(b[p+4 : p+8]))
		p += 8
		if n < 0 || p+n > len(b) {
			n = len(b) - p
		}
		body := b[p : p+n]
		switch id {
		case "fmt ":
			if len(body) >= 16 {
				tag = binary.LittleEndian.Uint16(body[0:])
				ch = binary.LittleEndian.Uint16(body[2:])
				rate = binary.LittleEndian.Uint32(body[4:])
				bits = binary.LittleEndian.Uint16(body[14:])
				if tag == 0xFFFE && len(body) >= 26 {
					tag = binary.LittleEndian.Uint16(body[24:])
				}
			}
		case "data":
			data = body
		}
		p += n + n&1
	}
	bps := int(bits) / 8
	if ch == 0 || rate == 0 || bps == 0 || data == nil {
		return nil
	}
	frames := len(data) / (bps * int(ch))
	if frames == 0 {
		return nil
	}
	src := make([]float32, frames*2)
	for i := 0; i < frames; i++ {
		var lr [2]float32
		for c := 0; c < int(ch) && c < 2; c++ {
			o := (i*int(ch) + c) * bps
			lr[c] = wavSample(data[o:o+bps], tag, bits)
		}
		if ch == 1 {
			lr[1] = lr[0]
		}
		src[2*i], src[2*i+1] = lr[0], lr[1]
	}
	if rate == outRate {
		return src
	}
	n := int(int64(frames) * outRate / int64(rate))
	out := make([]float32, n*2)
	step := float64(rate) / outRate
	for i := 0; i < n; i++ {
		x := float64(i) * step
		j := int(x)
		f := float32(x - float64(j))
		k := j + 1
		if k >= frames {
			k = frames - 1
		}
		for c := 0; c < 2; c++ {
			out[2*i+c] = src[2*j+c]*(1-f) + src[2*k+c]*f
		}
	}
	return out
}

func wavSample(s []byte, tag, bits uint16) float32 {
	switch {
	case tag == 3 && bits == 32:
		return math.Float32frombits(binary.LittleEndian.Uint32(s))
	case bits == 8:
		return (float32(s[0]) - 128) / 128
	case bits == 16:
		return float32(int16(binary.LittleEndian.Uint16(s))) / 32768
	case bits == 24:
		v := int32(uint32(s[0])<<8|uint32(s[1])<<16|uint32(s[2])<<24) >> 8
		return float32(v) / 8388608
	case bits == 32:
		return float32(int32(binary.LittleEndian.Uint32(s))) / 2147483648
	}
	return 0
}
