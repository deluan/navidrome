package tests

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

func NewMockFFmpeg(data string) *MockFFmpeg {
	return &MockFFmpeg{Reader: strings.NewReader(data)}
}

type MockFFmpeg struct {
	io.Reader
	lock   sync.Mutex
	closed atomic.Bool
	Error  error
	cover  *bytes.Buffer
}

func (ff *MockFFmpeg) Transcode(_ context.Context, _, _ string, _ int) (f io.ReadCloser, err error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff, nil
}

func (ff *MockFFmpeg) ExtractImage(context.Context, string) (*bytes.Buffer, error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff.cover, nil
}

func (ff *MockFFmpeg) Probe(context.Context, []string) (string, error) {
	if ff.Error != nil {
		return "", ff.Error
	}
	return "", nil
}
func (ff *MockFFmpeg) CmdPath() (string, error) {
	if ff.Error != nil {
		return "", ff.Error
	}
	return "ffmpeg", nil
}

func (ff *MockFFmpeg) Read(p []byte) (n int, err error) {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	return ff.Reader.Read(p)
}

func (ff *MockFFmpeg) Close() error {
	ff.closed.Store(true)
	return nil
}

func (ff *MockFFmpeg) IsClosed() bool {
	return ff.closed.Load()
}
