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

package main

import (
	"fmt"
	"time"

	"github.com/lni/pothosf"
)

// this is a toy test client, provided as an example
// it also print out various latencies and dispersions for visualization
func main() {
	client, err := pothosf.NewClient(pothosf.DefaultLockPath, pothosf.DefaultShmKey)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	start := time.Now()
	for {
		time.Sleep(100 * time.Millisecond)
		st := time.Now()
		ut, err := client.GetUnixTime()
		cost := time.Since(st)
		tt := time.Since(start)
		if err == pothosf.ErrStopped {
			fmt.Printf("clockd stopped\n")
			continue
		}
		if err == pothosf.ErrNotReady {
			fmt.Printf("clockd is not ready yet\n")
			continue
		}
		fmt.Printf("%g %d %d\n",
			float64(tt.Milliseconds())/3600000.0,
			ut.Dispersion,
			cost.Microseconds())
	}
}
