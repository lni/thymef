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

package gnomon

import (
	"encoding/binary"
	"errors"
	"os"

	"github.com/gen2brain/shm"
)

const (
	// Path of the lock file.
	DefaultLockPath string = "clockd.client.lock"
	// Key used for shared memory communication with clockd.
	DefaultShmKey int = 55356
	// buffer size of the shared memory.
	ClientInfoSharedMemoryBufferSize int   = 48
	staleThresholdNanoseconds        int64 = 300000000
)

var (
	// Encoder used for content stored in the shared memory region.
	Encoder = binary.BigEndian
)

var (
	// ErrNotReady indicates that clockd is not ready yet.
	ErrNotReady = errors.New("bounded time service not ready")
	// ErrStopped indicates that clockd unexpectedly stopped, e.g. crashed.
	ErrStopped = errors.New("bounded time service stopped")
)

// ClientInfo contains details exposed by clockd. Applications shouldn't be
// accessing any fields. All fields are in Unix time.
type ClientInfo struct {
	Valid      bool
	Locked     bool
	Count      uint16
	Dispersion uint64
	Sec        uint64
	NSec       uint32
}

func (c *ClientInfo) Marshal(buf []byte) ([]byte, error) {
	if len(buf) < 24 {
		panic("invalid buffer length")
	}

	if c.Valid {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	if c.Locked {
		buf[1] = 1
	} else {
		buf[1] = 0
	}

	Encoder.PutUint16(buf[2:], c.Count)
	Encoder.PutUint64(buf[4:], c.Dispersion)
	Encoder.PutUint64(buf[12:], c.Sec)
	Encoder.PutUint32(buf[20:], c.NSec)

	return buf[:24], nil
}

func UnmarshalClientInfo(data []byte, c *ClientInfo) error {
	if len(data) != 24 {
		panic("invalid input")
	}
	c.Valid = false
	if data[0] == 1 {
		c.Valid = true
	}
	c.Locked = false
	if data[1] == 1 {
		c.Locked = true
	}
	c.Count = Encoder.Uint16(data[2:])
	c.Dispersion = Encoder.Uint64(data[4:])
	c.Sec = Encoder.Uint64(data[12:])
	c.NSec = Encoder.Uint32(data[20:])

	return nil
}

// Client is the client used to get current bounded time. It is not thread safe
// meaning you shouldn't be using the same client concurrently from multiple
// threads.
type Client struct {
	lockPath string
	shmKey   int
	buf      []byte
	data     []byte
	mutex    *Semaphore
	shmID    int

	last struct {
		count uint16
		time  UnixTime
	}

	resetRequired bool
}

// NewClient creates a new Client instance.
func NewClient(lockPath string, shmKey int) (*Client, error) {
	c := &Client{
		lockPath: lockPath,
		shmKey:   shmKey,
		buf:      make([]byte, ClientInfoSharedMemoryBufferSize),
	}
	if err := reset(c); err != nil {
		return nil, err
	}

	return c, nil
}

// Close closes the client instance.
func (c *Client) Close() (err error) {
	if c.data != nil {
		err = FirstError(err, shm.Dt(c.data))
		c.data = nil
	}
	if c.mutex != nil {
		err = FirstError(err, c.mutex.Close())
		c.mutex = nil
	}

	return err
}

// GetUnixTime returns the UnixTime instance that presents the current time
// with associated uncertainty.
func (c *Client) GetUnixTime() (UnixTime, error) {
	data, sec, nsec, err := c.read()
	if err != nil {
		c.resetRequired = true
		return UnixTime{}, err
	}
	info := ClientInfo{}
	if err := UnmarshalClientInfo(data, &info); err != nil {
		panic(err)
	}
	if !info.Valid || !info.Locked {
		c.resetRequired = true
		return UnixTime{}, ErrNotReady
	}

	ut := UnixTime{
		Sec:        sec,
		NSec:       nsec,
		Dispersion: getDispersion(info, sec, nsec),
	}
	if c.updateStaled(ut, info.Count) {
		c.resetRequired = true
		return UnixTime{}, ErrStopped
	}
	if c.last.count != info.Count {
		c.last.count = info.Count
		c.last.time = ut
	}

	return ut, nil
}

func (c *Client) updateStaled(ut UnixTime, count uint16) bool {
	if c.last.count != count {
		return false
	}
	if c.last.time.IsEmpty() {
		return false
	}

	return ut.Sub(c.last.time) > staleThresholdNanoseconds
}

func reset(c *Client) error {
	_ = c.Close()

	m, err := NewSemaphore(c.lockPath, uint32(os.O_RDWR), 1)
	if err != nil {
		return err
	}
	shmID, err := shm.Get(c.shmKey, ClientInfoSharedMemoryBufferSize, shm.IPC_CREAT|0600)
	if err != nil {
		return err
	}
	data, err := shm.At(shmID, 0, 0)
	if err != nil {
		return err
	}

	c.mutex = m
	c.shmID = shmID
	c.data = data

	return nil
}

func (c *Client) tryReset() error {
	if c.resetRequired {
		c.resetRequired = false
		if err := reset(c); err != nil {
			c.resetRequired = true
			return err
		}
	}

	return nil
}

func (c *Client) read() (data []byte, sec uint64, nsec uint32, err error) {
	if err := c.tryReset(); err != nil {
		return nil, 0, 0, err
	}

	if err := c.mutex.Wait(); err != nil {
		return nil, 0, 0, err
	}
	defer func() {
		err = c.mutex.Post()
	}()
	sec, nsec = getSysClockTime()
	copy(c.buf, c.data)
	datalen := binary.BigEndian.Uint16(c.buf)
	if datalen == 0 {
		return nil, 0, 0, ErrNotReady
	}

	return c.buf[2 : 2+datalen], sec, nsec, nil
}
