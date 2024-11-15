// Copyright 2023-2024 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pothosf

import "time"

const (
	// MaxClockDrift is the absolute value of the max clock drift in ppb. 1e3ppm
	// or 1e6ppb is picked based on Microsoft's FarmV2 survey, which found that
	// most oscillators would drift significantly less than 200ppm. This leaves
	// us a 5x error budget.
	//
	// Note that it is highly unlikely to see such a large drift on a working
	// clock as the underlying digital system require the drift to be bounded
	// at a much smaller range, e.g. Intel NIC typically require a 30ppm max
	// drift for its SerDes and a 300ppm max drift for PCI-E, see Intel 82599
	// datasheet for more details. On the software side, kernel accepts 500ppm
	// maximum adjustment, meaning anything higher than that reflects a hardware
	// fault.
	MaxClockDrift int64 = 1000000
)

// UnixTime is the native time provided by clockd. It is used to represent
// current and future time with specified time uncertainties. Use time
// types from the stdlib or other 3rd libraries for generic time values. The
// Dispersion member is the uncertainty of the specified time meaning that
// for time t, the actual time is between [t-Dispersion, t+Dispersion] both
// inclusive.
type UnixTime struct {
	Sec        uint64
	NSec       uint32
	Dispersion uint64
}

// IsEmpty returns a boolean flag indicating whether the UnixTime instance is
// an empty value.
func (t *UnixTime) IsEmpty() bool {
	return t.Sec == 0 && t.NSec == 0
}

// Bounds returns the lower and upper limit of the time represented by the
// UnixTime instance.
func (t *UnixTime) Bounds() (uint64, uint64) {
	un := t.Sec*1e9 + uint64(t.NSec)
	return un - t.Dispersion, un + t.Dispersion
}

// Sub returns the time difference of (t - other) in nanoseconds.
func (t *UnixTime) Sub(other UnixTime) int64 {
	v := *t
	if other.NSec > v.NSec {
		v.NSec += 1e9
		v.Sec -= 1
	}

	sd := (int64(v.Sec) - int64(other.Sec)) * int64(1e9)
	nsd := int64(v.NSec) - int64(other.NSec)
	return sd + nsd
}

// GetClockUncertainty returns the dispersion introduced by the clock itself
// when we can not confirm whether it is broken or not. When there is a
// nanosecond worth of uncertain period, we multiply it with the MaxClockDrift
// to get the dispersion.
func GetClockUncertainty(nanosecond int64) uint64 {
	if nanosecond < 0 {
		panic("invalid value")
	}
	// 1s leads to 1ms uncertainty, that is Dispersion(1e12)
	fv := float64(nanosecond*MaxClockDrift) / float64(1e9)

	return uint64(fv)
}

func getDispersion(info ClientInfo, sec uint64, nsec uint32) uint64 {
	current := UnixTime{
		Sec:  sec,
		NSec: nsec,
	}
	ref := UnixTime{
		Sec:  info.Sec,
		NSec: info.NSec,
	}
	ns := current.Sub(ref)
	if ns < 0 {
		panic("invalid client info and clock time")
	}
	uct := GetClockUncertainty(ns)

	return info.Dispersion + uct
}

func getSysClockTime() (uint64, uint32) {
	t := time.Now()
	ns := t.UnixNano()
	sec := uint64(ns / 1e9)
	nsec := uint64(ns) - sec*1e9

	return sec, uint32(nsec)
}
