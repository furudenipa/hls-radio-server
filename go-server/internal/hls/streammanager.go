package hls

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrBufferFull = errors.New("segment buffer is full: maximum duration exceeded")
)

// PlaylistUpdater defines the interface for playlist update operations
type PlaylistUpdater interface {
	Update(segment) float64
}

// StreamManager defines the interface for managing HLS streams
type StreamManager interface {
	Run()
	Add(Content) error
	Kill()
	Pause()
	Resume()
}

type playlistManager struct {
	p PlaylistUpdater

	segQ       segmentsQueue
	killChan   chan struct{}
	pauseChan  chan struct{}
	resumeChan chan struct{}

	enoughBufferDuration float64

	statusMu sync.Mutex
	segQMu   sync.Mutex
	status   Status
}

func NewPlaylistManager(p PlaylistUpdater) *playlistManager {
	return &playlistManager{
		p: p,
		segQ: segmentsQueue{
			segments: make([]segment, 0),
		},
		killChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),

		enoughBufferDuration: 100.0,

		statusMu: sync.Mutex{},
		segQMu:   sync.Mutex{},
		status:   StatusDefault,
	}
}

func (m *playlistManager) Run() {
	m.statusMu.Lock()
	if m.status != StatusDefault {
		m.statusMu.Unlock()
		return
	}
	m.status = StatusStreaming
	m.statusMu.Unlock()

	var updatePlaylistChan <-chan time.Time
	updatePlaylistChan = time.After(time.Duration(250) * time.Millisecond)

	for {
		select {
		case <-updatePlaylistChan:
			m.statusMu.Lock()
			if m.status != StatusStreaming {
				updatePlaylistChan = time.After(time.Second)
				continue
			}
			m.statusMu.Unlock()

			m.segQMu.Lock()
			if seg, err := m.segQ.pop(); err == nil {
				fmt.Println("pop", seg.String()) //TEST
				wait := m.p.Update(seg)
				fmt.Println("wait", wait) //TEST
				if wait >= 0 {
					updatePlaylistChan = time.After(time.Duration(int(wait) * int(time.Second)))
				} else {
					//TODO: logging  wait is not positive
					updatePlaylistChan = time.After(time.Second)
				}
			} else {
				// TODO: logging  queue is empty or error
				updatePlaylistChan = time.After(time.Second)
			}
			m.segQMu.Unlock()

		case <-m.pauseChan:
			m.statusMu.Lock()
			if m.status == StatusStreaming {
				m.status = StatusPaused
			}
			m.statusMu.Unlock()

		case <-m.resumeChan:
			m.statusMu.Lock()
			if m.status == StatusPaused {
				m.status = StatusStreaming
			}
			m.statusMu.Unlock()

		case <-m.killChan:
			m.statusMu.Lock()
			m.status = StatusKilled
			m.statusMu.Unlock()
			// m.killChan = nil
			return
		}
	}
}

func (m *playlistManager) Add(c Content) error {
	m.segQMu.Lock()
	defer m.segQMu.Unlock()

	if m.segQ.totalDuration > m.enoughBufferDuration {
		return fmt.Errorf("buffer full (current: %.2f, max: %.2f): %w",
			m.segQ.totalDuration,
			m.enoughBufferDuration,
			ErrBufferFull)
	}

	segs := c.ToSegments()
	segs[0].discontinuity = true // 最初のセグメントにはDISCONTINUITYを入れる
	for _, seg := range segs {
		fmt.Println(seg.String()) //TEST
		m.segQ.push(seg)
	}
	return nil
}

func (m *playlistManager) Kill() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	if m.status != StatusKilled {
		close(m.killChan)
		m.status = StatusKilled
	}
}

func (m *playlistManager) Pause() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if m.status == StatusStreaming {
		m.pauseChan <- struct{}{}
	}
}

func (m *playlistManager) Resume() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if m.status == StatusPaused {
		m.resumeChan <- struct{}{}
	}
}
