package hls

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// テスト用のヘルパー関数
func setupTest(t *testing.T) (string, func()) {
	t.Helper()

	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "streammanager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// テスト用のm3u8ファイルを作成
	testM3U8 := []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:1.009,
segment1.ts
#EXTINF:1.008,
segment2.ts
`)

	if err := os.WriteFile(filepath.Join(tmpDir, "test.m3u8"), testM3U8, 0644); err != nil {
		t.Fatalf("failed to create test m3u8: %v", err)
	}

	// コンテンツディレクトリの作成とテストm3u8ファイルの生成
	for i := 1; i <= 10; i++ {
		contentDir := filepath.Join(tmpDir, "contents", "music", strconv.Itoa(i))
		if err := os.MkdirAll(contentDir, 0755); err != nil {
			t.Fatalf("failed to create content dir: %v", err)
		}

		// テスト用のm3u8ファイルを各コンテンツディレクトリに作成
		contentM3U8 := []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.0,
segment1.ts
`)
		if err := os.WriteFile(filepath.Join(contentDir, strconv.Itoa(i)+".m3u8"), contentM3U8, 0644); err != nil {
			t.Fatalf("failed to create content m3u8: %v", err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// テスト用の操作タイプ
type opType int

const (
	opRun opType = iota
	opAdd
	opStop
)

// テスト用の操作
type testOp struct {
	opType  opType
	content content
	wait    time.Duration
}

func TestNewStreamM3U8Manager(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name       string
		baseDir    string
		streamPath string
		wantErr    bool
	}{
		{
			name:       "正常系",
			baseDir:    tmpDir,
			streamPath: filepath.Join(tmpDir, "stream.m3u8"),
			wantErr:    false,
		},
		{
			name:       "不正なベースディレクトリ",
			baseDir:    "/nonexistent",
			streamPath: filepath.Join(tmpDir, "stream.m3u8"),
			wantErr:    false, // 初期化時点ではエラーにならない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStreamM3U8Manager(tt.baseDir, tt.streamPath)
			if (manager == nil) != tt.wantErr {
				t.Errorf("NewStreamM3U8Manager() error = %v, wantErr %v", manager == nil, tt.wantErr)
				return
			}

			if manager != nil {
				if manager.baseDir != tt.baseDir {
					t.Errorf("baseDir = %v, want %v", manager.baseDir, tt.baseDir)
				}
				if manager.streamFilePath != tt.streamPath {
					t.Errorf("streamFilePath = %v, want %v", manager.streamFilePath, tt.streamPath)
				}
				if manager.updateChan == nil {
					t.Error("updateChan is nil")
				}
				if manager.stopChan == nil {
					t.Error("stopChan is nil")
				}
			}
		})
	}
}

func TestStreamManagerOperations(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		operations  []testOp
		expectState Status
		expectErr   bool
	}{
		{
			name: "基本的な操作シーケンス",
			operations: []testOp{
				{opType: opRun},
				{opType: opAdd, content: content(*NewAudioContent(1, 10))},
				{opType: opAdd, content: content(*NewAudioContent(2, 10))},
				{opType: opStop},
			},
			expectState: StatusStopped,
			expectErr:   false,
		},
		{
			name: "早期Stop",
			operations: []testOp{
				{opType: opRun},
				{opType: opStop},
			},
			expectState: StatusStopped,
			expectErr:   false,
		},
		{
			name: "不正なコンテンツ",
			operations: []testOp{
				{opType: opRun},
				{opType: opAdd, content: content(*NewAudioContent(-1, -1))}, // 不正なパラメータ
				{opType: opStop},
			},
			expectState: StatusStopped,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStreamM3U8Manager(tmpDir, filepath.Join(tmpDir, "stream.m3u8"))

			var wg sync.WaitGroup
			for _, op := range tt.operations {
				switch op.opType {
				case opRun:
					wg.Add(1)
					go func() {
						defer wg.Done()
						manager.Run()
					}()
					time.Sleep(100 * time.Millisecond) // Run()の開始を待つ
				case opAdd:
					manager.Add(op.content)
					if op.wait > 0 {
						time.Sleep(op.wait)
					}
				case opStop:
					manager.Stop()
				}
			}
			wg.Wait()

			if manager.status != tt.expectState {
				t.Errorf("status = %v, want %v", manager.status, tt.expectState)
			}
		})
	}
}

func TestStreamManagerConcurrency(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		numAdd      int
		addDelay    time.Duration
		stopDelay   time.Duration
		expectState Status
	}{
		{
			name:        "複数の同時Add",
			numAdd:      5,
			addDelay:    1 * time.Millisecond,
			stopDelay:   10 * time.Millisecond,
			expectState: StatusStopped,
		},
		{
			name:        "高頻度のAdd",
			numAdd:      10,
			addDelay:    1 * time.Millisecond,
			stopDelay:   10 * time.Millisecond,
			expectState: StatusStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStreamM3U8Manager(tmpDir, filepath.Join(tmpDir, "stream.m3u8"))

			// Run goroutine
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				manager.Run()
			}()

			// Add operations
			for i := 0; i < tt.numAdd; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					time.Sleep(tt.addDelay)
					manager.Add(content(*NewAudioContent(id+1, 10)))
				}(i)
			}

			// Stop after delay
			time.Sleep(tt.stopDelay)
			manager.Stop()
			wg.Wait()

			if manager.status != tt.expectState {
				t.Errorf("status = %v, want %v", manager.status, tt.expectState)
			}
		})
	}
}

func TestStreamManagerEdgeCases(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		operation   func(*m3u8Manager)
		expectState Status
	}{
		{
			name: "Stop前のAdd",
			operation: func(m *m3u8Manager) {
				m.Add(content(*NewAudioContent(1, 10)))
				// 状態チェックのために少し待つ
				time.Sleep(100 * time.Millisecond)
			},
			expectState: StatusStreaming,
		},
		{
			name: "二重Stop",
			operation: func(m *m3u8Manager) {
				// Stopを呼び出して完全に停止するまで待つ
				m.Stop()
				time.Sleep(100 * time.Millisecond)

				// 2回目のStop
				m.Stop()
			},
			expectState: StatusStopped,
		},
		{
			name: "Stop後のAdd",
			operation: func(m *m3u8Manager) {
				m.Stop()
				// Stop後のAddはエラーにならないが、処理されない
				m.Add(content(*NewAudioContent(1, 10)))
			},
			expectState: StatusStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStreamM3U8Manager(tmpDir, filepath.Join(tmpDir, "stream.m3u8"))

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				manager.Run()
			}()

			time.Sleep(100 * time.Millisecond) // Run()の開始を待つ
			tt.operation(manager)

			if tt.expectState == StatusStopped {
				wg.Wait()
			} else {
				// StatusStreamingを期待する場合は、即座にステータスをチェック
				if manager.status != tt.expectState {
					t.Errorf("status = %v, want %v", manager.status, tt.expectState)
				}
				manager.Stop()
				wg.Wait()
			}
		})
	}
}
