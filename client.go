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
	"encoding/json"
	"errors"
	"os"

	"github.com/alexflint/go-filemutex"
	"github.com/fabiokung/shm"
)

const (
	ClientInfoSharedMemoryBufferSize int = 256
	DefaultLockPath                      = "/tmp/clockd.client.lock"
	DefaultShmPath                       = "clockd.shm"
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

func (c *ClientInfo) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func UnmarshalClientInfo(data []byte, c *ClientInfo) error {
	return json.Unmarshal(data, c)
}

// Client is the client used to get current bounded time. It is not thread safe
// meaning you shouldn't be using the same client concurrently from multiple
// threads.
type Client struct {
	lb      []byte
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
		lb:      make([]byte, 2),
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
	info, sec, nsec, err := c.read()
	if err != nil {
		return UnixTime{}, err
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

func (c *Client) read() (ClientInfo, uint64, uint32, error) {
	var r1 ClientInfo
	var r2 ClientInfo

	for {
		buf, err := c.readFromShm()
		if err != nil {
			return ClientInfo{}, 0, 0, err
		}
		if err := UnmarshalClientInfo(buf, &r1); err != nil {
			panic(err)
		}
		sec, nsec := getSysClockTime()
		buf, err = c.readFromShm()
		if err != nil {
			return ClientInfo{}, 0, 0, err
		}
		if err := UnmarshalClientInfo(buf, &r2); err != nil {
			panic(err)
		}
		if r1.Count == r2.Count {
			return r1, sec, nsec, nil
		}
	}

	panic("not suppose to reach here")
}

func (c *Client) readFromShm() ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, err := c.shmfile.ReadAt(c.lb, 0); err != nil {
		return nil, err
	}
	datalen := binary.BigEndian.Uint16(c.lb)
	if _, err := c.shmfile.ReadAt(c.buf[:datalen], 2); err != nil {
		return nil, err
	}

	return c.buf[:datalen], nil
}
