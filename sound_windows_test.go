package main

import (
	"encoding/binary"
	"math"
	"testing"
)

// decodeWav must keep the source loudness: the peak of the decoded
// samples equals the peak of the source samples, within one LSB when
// the file is already at the output rate (no resampling).
func TestDecodeWavKeepsPeak(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"hover", hoverWav},
		{"appear", appearWav},
		{"drop", dropWav},
	} {
		srcPeak, rate := wavSourcePeak(t, tc.raw)
		d := decodeWav(tc.raw)
		if d == nil {
			t.Fatalf("%s: decodeWav returned nil", tc.name)
		}
		if len(d) == 0 {
			t.Fatalf("%s: decodeWav returned no samples", tc.name)
		}
		outPeak := float32(0)
		for _, s := range d {
			if s > outPeak {
				outPeak = s
			}
			if -s > outPeak {
				outPeak = -s
			}
		}
		lsb := 1.0 / 32768
		if outPeak > srcPeak+float32(lsb) {
			t.Errorf("%s: decoded peak %g above the source peak %g", tc.name, outPeak, srcPeak)
		}
		if rate == outRate && math.Abs(float64(outPeak-srcPeak)) > lsb {
			t.Errorf("%s: rate unchanged (%d), decoded peak %g != source peak %g", tc.name, rate, outPeak, srcPeak)
		}
	}
}

// wavSourcePeak parses the WAV on its own and returns the peak of its
// samples, converted to floats the way decodeWav converts them, plus
// the file's sample rate.
func wavSourcePeak(t *testing.T, b []byte) (float32, uint32) {
	t.Helper()
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		t.Fatal("not a RIFF/WAVE file")
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
			}
		case "data":
			data = body
		}
		p += n + n&1
	}
	bps := int(bits) / 8
	if ch == 0 || rate == 0 || bps == 0 || data == nil {
		t.Fatal("unsupported WAV")
	}
	frames := len(data) / (bps * int(ch))
	peak := float32(0)
	for i := 0; i < frames; i++ {
		for c := 0; c < int(ch) && c < 2; c++ {
			o := (i*int(ch) + c) * bps
			v := wavSample(data[o:o+bps], tag, bits)
			if v > peak {
				peak = v
			}
			if -v > peak {
				peak = -v
			}
		}
	}
	return peak, rate
}
