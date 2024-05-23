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

	"github.com/alexflint/go-filemutex"
	"github.com/fabiokung/shm"
)

const (
	ClientInfoSharedMemoryBufferSize int = 48
	DefaultLockPath                      = "/tmp/clockd.client.lock"
	DefaultShmPath                       = "clockd.shm"
)

var (
	encoder = binary.BigEndian
)

var (
	ErrNotReady = errors.New("time service not ready")
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

	encoder.PutUint16(buf[2:], c.Count)
	encoder.PutUint64(buf[4:], c.Dispersion)
	encoder.PutUint64(buf[12:], c.Sec)
	encoder.PutUint32(buf[20:], c.NSec)

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
	c.Count = encoder.Uint16(data[2:])
	c.Dispersion = encoder.Uint64(data[4:])
	c.Sec = encoder.Uint64(data[12:])
	c.NSec = encoder.Uint32(data[20:])

	return nil
}

// Client is the client used to get current bounded time. It is not thread safe
// meaning you shouldn't be using the same client concurrently from multiple
// threads.
type Client struct {
	buf     []byte
	mutex   *filemutex.FileMutex
	shmfile *os.File
}

// NewClient creates a new Client instance.
func NewClient(lockPath string, shmPath string) (*Client, error) {
	m, err := filemutex.New(lockPath)
	if err != nil {
		return nil, err
	}
	file, err := shm.Open(shmPath, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	e := &Client{
		mutex:   m,
		shmfile: file,
		buf:     make([]byte, ClientInfoSharedMemoryBufferSize),
	}
	return e, nil
}

// Close closes the client instance.
func (c *Client) Close() (err error) {
	err = FirstError(err, c.shmfile.Close())

	return err
}

// GetUnixTime returns the UnixTime instance that presents the current time
// with associated uncertainty.
func (c *Client) GetUnixTime() (UnixTime, error) {
	data, sec, nsec, err := c.read()
	if err != nil {
		return UnixTime{}, err
	}
	info := ClientInfo{}
	if err := UnmarshalClientInfo(data, &info); err != nil {
		panic(err)
	}
	if !info.Valid || !info.Locked {
		return UnixTime{}, ErrNotReady
	}

	return UnixTime{
		Sec:        sec,
		NSec:       nsec,
		Dispersion: getDispersion(info, sec, nsec),
	}, nil
}

func (c *Client) read() ([]byte, uint64, uint32, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	sec, nsec := getSysClockTime()
	if _, err := c.shmfile.ReadAt(c.buf, 0); err != nil {
		return nil, 0, 0, err
	}
	datalen := binary.BigEndian.Uint16(c.buf)

	return c.buf[2 : 2+datalen], sec, nsec, nil
}
