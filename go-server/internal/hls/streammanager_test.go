package hls

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// モックの実装
type mockPlaylist struct {
	mu          sync.Mutex
	updateCount int
	lastSegment *segment
	updateDelay time.Duration
	err         error
}

func newMockPlaylist() *mockPlaylist {
	return &mockPlaylist{
		mu:          sync.Mutex{},
		updateCount: 0,
	}
}

func (m *mockPlaylist) Update(seg segment) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return 0
	}

	if m.updateDelay > 0 {
		time.Sleep(m.updateDelay)
	}

	m.updateCount++
	m.lastSegment = &seg
	return 10.0
}

func (m *mockPlaylist) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockPlaylist) SetUpdateDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateDelay = d
}

func (m *mockPlaylist) GetUpdateCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCount
}

func (m *mockPlaylist) GetLastSegment() *segment {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastSegment
}

// モックコンテンツ
type mockContent struct {
	id       int
	segments []segment
}

func newMockContent(segments []segment) mockContent {
	return mockContent{
		segments: segments,
		id:       1,
	}
}

func (m mockContent) ToSegments() []segment {
	return m.segments
}

func (m mockContent) SourcePath() string {
	return "/test/content/" + strconv.Itoa(m.id) + ".m3u8"
}

func (m mockContent) UrlPath() string {
	return "/contents/test/" + strconv.Itoa(m.id) + ".m3u8"
}

func (m mockContent) SegmentLocalToGlobal(seg segment) segment {
	seg.uri = "/contents/test/" + strconv.Itoa(m.id) + "/" + seg.uri
	return seg
}

func (m mockContent) ToStreamFilePath(baseDir string) string {
	return filepath.Join(baseDir, "contents", "test", strconv.Itoa(m.id)+".m3u8")
}

// テストコンテキスト
type testContext struct {
	ctx      context.Context
	cancel   context.CancelFunc
	manager  *playlistManager
	playlist *mockPlaylist
	wg       sync.WaitGroup
	errChan  chan error
}

func newTestContext(t *testing.T) *testContext {
	ctx, cancel := context.WithCancel(context.Background())
	playlist := newMockPlaylist()
	manager := NewPlaylistManager(playlist)

	return &testContext{
		ctx:      ctx,
		cancel:   cancel,
		manager:  manager,
		playlist: playlist,
		errChan:  make(chan error, 1),
	}
}

func (tc *testContext) cleanup() {
	tc.cancel()
	tc.wg.Wait()
	close(tc.errChan)
}

func (tc *testContext) runWithTimeout(t *testing.T, timeout time.Duration, f func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- f()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		t.Fatal("test timed out")
		return nil
	}
}

// テストケース構造体
type testCase struct {
	name    string
	setup   func(*testContext)
	run     func(*testing.T, *testContext) error
	verify  func(*testing.T, *testContext)
	timeout time.Duration
}

func runTestCase(t *testing.T, tc testCase) {
	ctx := newTestContext(t)
	defer ctx.cleanup()

	if tc.setup != nil {
		tc.setup(ctx)
	}

	if err := ctx.runWithTimeout(t, tc.timeout, func() error {
		return tc.run(t, ctx)
	}); err != nil {
		t.Errorf("test failed: %v", err)
		return
	}

	if tc.verify != nil {
		tc.verify(t, ctx)
	}
}

// 基本的なテストケース
func TestNewPlaylistManager(t *testing.T) {
	tests := []testCase{
		{
			name: "basic_initialization",
			run: func(t *testing.T, tc *testContext) error {
				if tc.manager == nil {
					return errors.New("manager should not be nil")
				}
				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				if tc.manager.status != StatusDefault {
					t.Errorf("initial status = %v, want %v", tc.manager.status, StatusDefault)
				}

				if tc.manager.enoughBufferDuration != 100.0 {
					t.Errorf("buffer duration = %v, want %v", tc.manager.enoughBufferDuration, 100.0)
				}
			},
			timeout: time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestPlaylistManager_Run(t *testing.T) {
	tests := []testCase{
		{
			name: "normal_operation",
			run: func(t *testing.T, tc *testContext) error {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond) // 状態変更を待つ

				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusStreaming {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusStreaming)
				}
			},
			timeout: time.Second,
		},
		{
			name: "already_running",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
			},
			run: func(t *testing.T, tc *testContext) error {
				tc.manager.Run() // 2回目の実行
				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusStreaming {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusStreaming)
				}
			},
			timeout: time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestPlaylistManager_Add(t *testing.T) {
	tests := []testCase{
		{
			name: "add_content_success",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
			},
			run: func(t *testing.T, tc *testContext) error {
				content := newMockContent([]segment{
					{duration: 10.0, uri: "test1.ts"},
					{duration: 10.0, uri: "test2.ts"},
				})
				return tc.manager.Add(content)
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.segQMu.Lock()
				qLen := len(tc.manager.segQ.segments)
				tc.manager.segQMu.Unlock()

				if qLen != 2 {
					t.Errorf("queue length = %v, want %v", qLen, 2)
				}
			},
			timeout: time.Second,
		},
		{
			name: "add_content_buffer_full",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)

				// バッファを満杯にする
				content := newMockContent([]segment{
					{duration: 60.0, uri: "test1.ts"},
					{duration: 60.0, uri: "test2.ts"},
				})
				if err := tc.manager.Add(content); err != nil {
					t.Errorf("failed to add content: %v", err)
				}
			},
			run: func(t *testing.T, tc *testContext) error {
				content := newMockContent([]segment{
					{duration: 10.0, uri: "test3.ts"},
				})
				err := tc.manager.Add(content)
				if !errors.Is(err, ErrBufferFull) {
					t.Errorf("expected ErrBufferFull but got: %v", err)
				}
				return nil
			},
			timeout: time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestPlaylistManager_Kill(t *testing.T) {
	tests := []testCase{
		{
			name: "kill_running_manager",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
			},
			run: func(t *testing.T, tc *testContext) error {
				tc.manager.Kill()
				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusKilled {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusKilled)
				}
			},
			timeout: time.Second,
		},
		{
			name: "kill_already_killed",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
				tc.manager.Kill()
			},
			run: func(t *testing.T, tc *testContext) error {
				tc.manager.Kill() // 2回目のKill
				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusKilled {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusKilled)
				}
			},
			timeout: time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestPlaylistManager_PauseResume(t *testing.T) {
	tests := []testCase{
		{
			name: "pause_resume_cycle",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
			},
			run: func(t *testing.T, tc *testContext) error {
				tc.manager.Pause()
				time.Sleep(100 * time.Millisecond)

				tc.manager.statusMu.Lock()
				if tc.manager.status != StatusPaused {
					tc.manager.statusMu.Unlock()
					return errors.New("manager should be paused")
				}
				tc.manager.statusMu.Unlock()

				tc.manager.Resume()
				time.Sleep(100 * time.Millisecond)

				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusStreaming {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusStreaming)
				}
			},
			timeout: time.Second,
		},
		{
			name: "pause_when_not_streaming",
			setup: func(tc *testContext) {
				// Runを実行せずにテスト
			},
			run: func(t *testing.T, tc *testContext) error {
				tc.manager.Pause()
				return nil
			},
			verify: func(t *testing.T, tc *testContext) {
				tc.manager.statusMu.Lock()
				defer tc.manager.statusMu.Unlock()

				if tc.manager.status != StatusDefault {
					t.Errorf("status = %v, want %v", tc.manager.status, StatusDefault)
				}
			},
			timeout: time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

func TestPlaylistManager_Concurrency(t *testing.T) {
	tests := []testCase{
		{
			name: "concurrent_adds",
			setup: func(tc *testContext) {
				go tc.manager.Run()
				time.Sleep(100 * time.Millisecond)
				tc.playlist.SetUpdateDelay(50 * time.Millisecond)
			},
			run: func(t *testing.T, tc *testContext) error {
				var wg sync.WaitGroup
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						content := newMockContent([]segment{
							{duration: 10.0, uri: "test1.ts"},
						})
						if err := tc.manager.Add(content); err != nil {
							tc.errChan <- err
						}
					}(i)
				}
				wg.Wait()

				select {
				case err := <-tc.errChan:
					return err
				default:
					return nil
				}
			},
			verify: func(t *testing.T, tc *testContext) {
				time.Sleep(300 * time.Millisecond) // 更新を待つ

				updateCount := tc.playlist.GetUpdateCount()
				if updateCount == 0 {
					t.Error("no updates occurred")
				}
			},
			timeout: 2 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}
