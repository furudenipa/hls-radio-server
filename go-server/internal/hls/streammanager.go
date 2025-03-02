package hls

import (
	"log/slog"
	"sync"
)

type StreamManager interface {
	Run()
	Add(content)
	Stop()
}

type m3u8Manager struct {
	stream *playlist

	updateChan     chan content
	stopChan       chan struct{}
	baseDir        string
	streamFilePath string

	mu sync.Mutex

	status Status
}

func NewStreamM3U8Manager(baseDir, streamFilePath string) *m3u8Manager {
	const maxConn = 10
	playlist, err := NewM3U8(streamFilePath)
	if err != nil {
		slog.Error("failed to create m3u8", "error", err)
	}
	return &m3u8Manager{
		stream:         playlist, // TODO:
		updateChan:     make(chan content, maxConn),
		stopChan:       make(chan struct{}),
		baseDir:        baseDir,
		streamFilePath: streamFilePath,
	}
}

func (m *m3u8Manager) Run() {
	defer close(m.updateChan)
	m.status = StatusStreaming
	for {
		select {
		case c := <-m.updateChan:
			if err := m.HandleUpdate(c); err != nil {
				slog.Error("faild to update stream", "err", err)
			}
		case <-m.stopChan:
			m.status = StatusStopped
			return
		}
	}
}

// idをもとにstream.m3u8を更新する
func (m *m3u8Manager) HandleUpdate(c content) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sourcePath := c.ToStreamFilePath(m.baseDir)
	return m.stream.SyncFromSource(sourcePath, m.streamFilePath, m.stopChan)
}

func (m *m3u8Manager) Add(c content) {
	m.updateChan <- c
}

func (m *m3u8Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status != StatusStopped {
		close(m.stopChan)
		m.status = StatusStopped
	}
}
