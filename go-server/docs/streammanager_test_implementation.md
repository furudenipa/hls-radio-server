# StreamManager Test Implementation Guide

## 1. テスト構造の概要

### 1.1 テストコンポーネント

```go
// モックインターフェース
type playlistMock interface {
    Update(segment) float64
    SetError(err error)
    SetUpdateDelay(d time.Duration)
    GetUpdateCount() int
    GetLastSegment() *segment
}

type contentMock interface {
    ToSegments() []segment
    SetError(err error)
}

// テストコンテキスト
type testContext struct {
    ctx        context.Context
    cancel     context.CancelFunc
    manager    *playlistManager
    playlist   playlistMock
    wg         sync.WaitGroup
    errChan    chan error
}
```

### 1.2 テストケース構造

```go
type testCase struct {
    name        string
    setup       func(*testContext)
    run         func(*testing.T, *testContext) error
    verify      func(*testing.T, *testContext)
    cleanup     func(*testContext)
    timeout     time.Duration
}
```

## 2. テストカバレッジ計画

### 2.1 基本機能テスト

1. 初期化テスト（TestNewPlaylistManager）
   - 基本設定の検証
   - チャネルの初期化確認
   - デフォルト状態の確認

2. ライフサイクルテスト（TestPlaylistManager_Lifecycle）
   - 正常系の動作確認
   - エラーからの回復
   - リソースのクリーンアップ

3. コンテンツ管理テスト（TestPlaylistManager_ContentManagement）
   - コンテンツの追加
   - バッファ制限の検証
   - エラー状態の処理

### 2.2 状態管理テスト

1. 状態遷移テスト
   - 各状態間の遷移検証
   - 無効な遷移のハンドリング
   - 状態変更の同期確認

2. ポーズ/レジューム機能
   - 一時停止と再開の動作
   - 状態変更のタイミング
   - イベント順序の検証

### 2.3 並行処理テスト

1. 同時実行テスト
   - 複数のAdd操作
   - 状態変更の競合
   - デッドロック防止

2. リソース管理テスト
   - メモリリークの防止
   - ゴルーチンの管理
   - チャネルのクリーンアップ

## 3. 実装の詳細

### 3.1 モックの実装

```go
type mockPlaylist struct {
    mu          sync.Mutex
    updateCount int
    lastSegment *segment
    updateDelay time.Duration
    err         error
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
```

### 3.2 テストヘルパー関数

```go
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
```

## 4. テストシナリオ

### 4.1 基本シナリオ

1. 正常系フロー
```go
// テストケース例
{
    name: "normal_operation",
    setup: func(tc *testContext) {
        // 通常の初期化
    },
    run: func(t *testing.T, tc *testContext) error {
        // 1. マネージャーの開始
        // 2. コンテンツの追加
        // 3. 更新の確認
        return nil
    },
    verify: func(t *testing.T, tc *testContext) {
        // 結果の検証
    },
    timeout: 5 * time.Second,
}
```

### 4.2 エラーシナリオ

1. エラー回復フロー
```go
{
    name: "error_recovery",
    setup: func(tc *testContext) {
        tc.playlist.SetError(errors.New("update error"))
    },
    run: func(t *testing.T, tc *testContext) error {
        // エラー発生と回復のシミュレーション
        return nil
    },
    verify: func(t *testing.T, tc *testContext) {
        // 回復状態の確認
    },
    timeout: 5 * time.Second,
}
```

## 5. 実装上の注意点

### 5.1 タイミング制御
- コンテキストによるタイムアウト管理
- 適切な待機時間の設定
- 環境依存の最小化

### 5.2 リソース管理
- すべてのゴルーチンの終了確認
- チャネルの適切なクローズ
- メモリリークの防止

### 5.3 テストの独立性
- テストケース間の状態分離
- クリーンアップの確実な実行
- グローバル状態の回避

### 5.4 エラー処理
- エラーケースの明示的なテスト
- エラー状態からの回復確認
- エラーメッセージの検証

## 6. 将来の改善点

1. パフォーマンステストの追加
   - 負荷テストシナリオ
   - メモリ使用量の監視
   - 実行時間の最適化

2. テストカバレッジの拡充
   - エッジケースの追加
   - 異常系テストの強化
   - 境界値テストの拡充

3. テストユーティリティの改善
   - ヘルパー関数の汎用化
   - アサーション関数の追加
   - テストデータ生成の効率化
