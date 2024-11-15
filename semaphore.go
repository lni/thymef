// MIT License
//
// Copyright (c) 2016 Crown Equipment Corp.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
// https://github.com/aka-mj/go-semaphore
//
// modified from https://github.com/aka-mj/go-semaphore
//
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

import (
	"syscall"
	"unsafe"
)

// #cgo LDFLAGS: -pthread
// #include <stdlib.h>
// #include <fcntl.h>
// #include <sys/stat.h>
// #include <sys/types.h>
// #include <semaphore.h>
// #include <time.h>
// #ifndef GO_SEM_LIB_
// #define GO_SEM_LIB_
// sem_t* Go_sem_open(const char *name, int oflag, mode_t mode, unsigned int value)
// {
//		return sem_open(name, oflag, mode, value);
// }
// #endif
import "C"

type Semaphore struct {
	sem  *C.sem_t //semaphore returned by sem_open
	name string   //name of semaphore
}

// Open creates a new POSIX semaphore or opens an existing semaphore.
// The semaphore is identified by name. The mode argument specifies the permissions
// to be placed on the new semaphore. The value argument specifies the initial
// value for the new semaphore. If the named semaphore already exist, mode and
// value are ignored.
// For details see sem_overview(7).
func NewSemaphore(name string, mode, value uint32) (*Semaphore, error) {
	n := C.CString(name)
	sem, err := C.Go_sem_open(n, syscall.O_CREAT, C.mode_t(mode), C.uint(value))
	C.free(unsafe.Pointer(n))
	if sem == nil {
		return nil, err
	}

	return &Semaphore{sem: sem, name: name}, nil
}

// Close closes the named semaphore, allowing any resources that the system has
// allocated to the calling process for this semaphore to be freed.
func (s *Semaphore) Close() error {
	ret, err := C.sem_close(s.sem)
	if ret != 0 {
		return err
	}

	return nil
}

// Post increments the semaphore.
func (s *Semaphore) Post() error {
	ret, err := C.sem_post(s.sem)
	if ret != 0 {
		return err
	}

	return nil
}

// Wait decrements the semaphore. If the semaphore's value is greater than zero,
// then the decrement proceeds, and the function returns, immediately. If the
// semaphore currently has the value zero, then the call blocks until either
// it becomes possible to perform the decrement, or a signal interrupts the call.
func (s *Semaphore) Wait() error {
	ret, err := C.sem_wait(s.sem)
	if ret != 0 {
		return err
	}

	return nil
}

// Unlink removes the named semaphore. The semaphore name is removed immediately.
// The semaphore is destroyed once all other processes that have the semaphore
// open close it.
func (s *Semaphore) Unlink() error {
	name := C.CString(s.name)
	ret, err := C.sem_unlink(name)
	C.free(unsafe.Pointer(name))
	if ret != 0 {
		return err
	}

	return nil
}
