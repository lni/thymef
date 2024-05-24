// Copyright 2017-2021 Lei Ni (nilei81@gmail.com) and other contributors.
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

package gnomon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		sec    uint64
		nsec   uint32
		result bool
	}{
		{0, 0, true},
		{1, 0, false},
		{0, 1, false},
		{1, 1, false},
	}

	for idx, tt := range tests {
		ut := UnixTime{
			Sec:  tt.sec,
			NSec: tt.nsec,
		}
		assert.Equal(t, tt.result, ut.IsEmpty(), idx)
	}
}

func TestRange(t *testing.T) {
	ut := UnixTime{
		Sec:        2,
		NSec:       100200,
		Dispersion: 8,
	}

	lower, upper := ut.Bounds()
	assert.Equal(t, uint64(2000100192), lower)
	assert.Equal(t, uint64(2000100208), upper)
}

func TestUnixTimeSub(t *testing.T) {
	tests := []struct {
		sec    uint64
		nsec   uint32
		oSec   uint64
		oNsec  uint32
		result int64
	}{
		{1, 0, 1, 0, 0},
		{1, 100, 1, 0, 100},
		{1, 100, 1, 200, -100},
		{1, 0, 0, 0, 1e9},
		{0, 100, 1, 100, -1e9},
		{2, 100, 1, 200, 1e9 - 100},
	}

	for idx, tt := range tests {
		v := UnixTime{
			Sec:  tt.sec,
			NSec: tt.nsec,
		}
		o := UnixTime{
			Sec:  tt.oSec,
			NSec: tt.oNsec,
		}
		assert.Equal(t, tt.result, v.Sub(o), idx)
	}
}

func TestGetClockUncertainty(t *testing.T) {
	tests := []struct {
		val    int64
		result uint64
	}{
		{0, 0},
		{1e9, 1000000},
		{1e8, 100000},
		{1e3, 1},
	}

	for idx, tt := range tests {
		assert.Equal(t, tt.result, GetClockUncertainty(tt.val), idx)
	}
}

func TestGetDispersionWithInvalidInput(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fail()
		}
	}()

	info := ClientInfo{
		Sec:  2,
		NSec: 0,
	}
	getDispersion(info, 1, 0)
}

func TestGetDispersion(t *testing.T) {
	tests := []struct {
		sec        uint64
		nsec       uint32
		dispersion uint64
		oSec       uint64
		oNsec      uint32
		result     uint64
	}{
		{1, 0, 1, 1, 0, 1},
		{1, 0, 100, 2, 0, 1e6 + 100},
		{1, 1000, 100, 1, 2000, 101},
	}

	for idx, tt := range tests {
		info := ClientInfo{
			Sec:        tt.sec,
			NSec:       tt.nsec,
			Dispersion: tt.dispersion,
		}
		result := getDispersion(info, tt.oSec, tt.oNsec)
		assert.Equal(t, tt.result, result, idx)
	}
}
