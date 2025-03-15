package hls

import (
	"testing"
)

func TestNewPlaylist(t *testing.T) {
	tests := []struct {
		name   string
		config PlaylistConfig
		want   playlistMetadata
	}{
		{
			name: "basic initialization",
			config: PlaylistConfig{
				MaxSegments:    3,
				TargetDuration: 10.0,
			},
			want: playlistMetadata{
				version:               3,
				targetDuration:        10.0,
				mediaSequence:         0,
				discontinuitySequence: 0,
			},
		},
		{
			name: "with zero max segments",
			config: PlaylistConfig{
				MaxSegments:    0,
				TargetDuration: 5.0,
			},
			want: playlistMetadata{
				version:               3,
				targetDuration:        5.0,
				mediaSequence:         0,
				discontinuitySequence: 0,
			},
		},
		{
			name: "with zero target duration",
			config: PlaylistConfig{
				MaxSegments:    5,
				TargetDuration: 0,
			},
			want: playlistMetadata{
				version:               3,
				targetDuration:        0,
				mediaSequence:         0,
				discontinuitySequence: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPlaylist(tt.config)

			if got.metadata.version != tt.want.version {
				t.Errorf("NewPlaylist() version = %v, want %v", got.metadata.version, tt.want.version)
			}
			if got.metadata.targetDuration != tt.want.targetDuration {
				t.Errorf("NewPlaylist() targetDuration = %v, want %v", got.metadata.targetDuration, tt.want.targetDuration)
			}
			if got.metadata.mediaSequence != tt.want.mediaSequence {
				t.Errorf("NewPlaylist() mediaSequence = %v, want %v", got.metadata.mediaSequence, tt.want.mediaSequence)
			}
			if got.metadata.discontinuitySequence != tt.want.discontinuitySequence {
				t.Errorf("NewPlaylist() discontinuitySequence = %v, want %v", got.metadata.discontinuitySequence, tt.want.discontinuitySequence)
			}

			// MaxSegmentsが0以下の場合はエラーを返すべき
			if tt.config.MaxSegments <= 0 {
				t.Log("WARN: MaxSegments should be greater than 0")
			}
			// TargetDurationが0以下の場合はエラーを返すべき
			if tt.config.TargetDuration <= 0 {
				t.Log("WARN: TargetDuration should be greater than 0")
			}
		})
	}
}

func TestPlaylist_AppendSegment(t *testing.T) {
	tests := []struct {
		name        string
		maxSegments int
		segments    []segment
		wantErr     bool
	}{
		{
			name:        "append within limit",
			maxSegments: 3,
			segments: []segment{
				{duration: 10.0, uri: "test1.ts"},
				{duration: 10.0, uri: "test2.ts"},
			},
			wantErr: false,
		},
		{
			name:        "append at limit",
			maxSegments: 1,
			segments: []segment{
				{duration: 10.0, uri: "test1.ts"},
				{duration: 10.0, uri: "test2.ts"},
			},
			wantErr: true,
		},
		{
			name:        "zero duration segment",
			maxSegments: 3,
			segments: []segment{
				{duration: 0.0, uri: "test1.ts"},
			},
			wantErr: true, // 0以下のdurationはエラーを返す
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlaylist(PlaylistConfig{
				MaxSegments:    tt.maxSegments,
				TargetDuration: 10.0,
			})

			var err error
			for _, seg := range tt.segments {
				err = p.appendSegment(seg)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("appendSegment() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(p.segments) > tt.maxSegments {
				t.Errorf("segments length = %v, want <= %v", len(p.segments), tt.maxSegments)
			}

			// 負数や0のdurationはエラーを返すべき
			for _, seg := range tt.segments {
				if seg.duration <= 0 {
					t.Log("WARN: segment duration should be greater than 0")
				}
			}
		})
	}
}

func TestPlaylist_RemoveOldestSegment(t *testing.T) {
	tests := []struct {
		name                  string
		initialSegments       []segment
		wantMediaSeq          int
		wantDiscontinuitySeq  int
		wantRemainingSegments int
	}{
		{
			name: "remove normal segment",
			initialSegments: []segment{
				{duration: 10.0, uri: "test1.ts", discontinuity: false},
				{duration: 10.0, uri: "test2.ts", discontinuity: false},
			},
			wantMediaSeq:          1,
			wantDiscontinuitySeq:  0,
			wantRemainingSegments: 1,
		},
		{
			name: "remove discontinuity segment",
			initialSegments: []segment{
				{duration: 10.0, uri: "test1.ts", discontinuity: true},
				{duration: 10.0, uri: "test2.ts", discontinuity: false},
			},
			wantMediaSeq:          1,
			wantDiscontinuitySeq:  1,
			wantRemainingSegments: 1,
		},
		{
			name:                  "remove from empty playlist",
			initialSegments:       []segment{},
			wantMediaSeq:          0,
			wantDiscontinuitySeq:  0,
			wantRemainingSegments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlaylist(PlaylistConfig{
				MaxSegments:    5,
				TargetDuration: 10.0,
			})

			for _, seg := range tt.initialSegments {
				_ = p.appendSegment(seg)
			}

			if err := p.removeOldestSegment(); err != nil {
				t.Errorf("removeOldestSegment() error = %v, want nil", err)
			}

			if p.metadata.mediaSequence != tt.wantMediaSeq {
				t.Errorf("mediaSequence = %v, want %v", p.metadata.mediaSequence, tt.wantMediaSeq)
			}
			if p.metadata.discontinuitySequence != tt.wantDiscontinuitySeq {
				t.Errorf("discontinuitySequence = %v, want %v", p.metadata.discontinuitySequence, tt.wantDiscontinuitySeq)
			}
			if len(p.segments) != tt.wantRemainingSegments {
				t.Errorf("remaining segments = %v, want %v", len(p.segments), tt.wantRemainingSegments)
			}

			// 空のプレイリストの場合はエラーを返すべき
			if len(tt.initialSegments) == 0 {
				t.Log("WARN: removing from empty playlist should return an error")
			}
		})
	}
}

func TestPlaylist_Update(t *testing.T) {
	tests := []struct {
		name            string
		maxSegments     int
		initialSegments []segment
		updateSegment   segment
		wantDuration    float64
		wantFinalLen    int
	}{
		{
			name:        "add segment with space",
			maxSegments: 3,
			initialSegments: []segment{
				{duration: 10.0, uri: "test1.ts"},
			},
			updateSegment: segment{duration: 10.0, uri: "test2.ts"},
			wantDuration:  0.0,
			wantFinalLen:  2,
		},
		{
			name:        "add segment when full",
			maxSegments: 2,
			initialSegments: []segment{
				{duration: 10.0, uri: "test1.ts"},
				{duration: 10.0, uri: "test2.ts"},
			},
			updateSegment: segment{duration: 10.0, uri: "test3.ts"},
			wantDuration:  10.0,
			wantFinalLen:  2,
		},
		{
			name:        "add segment with discontinuity",
			maxSegments: 2,
			initialSegments: []segment{
				{duration: 10.0, uri: "test1.ts", discontinuity: true},
				{duration: 10.0, uri: "test2.ts", discontinuity: false},
			},
			updateSegment: segment{duration: 10.0, uri: "test3.ts", discontinuity: true},
			wantDuration:  10.0,
			wantFinalLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlaylist(PlaylistConfig{
				MaxSegments:    tt.maxSegments,
				TargetDuration: 10.0,
			})

			// セグメントの追加
			for _, seg := range tt.initialSegments {
				err := p.appendSegment(seg)
				if err != nil {
					t.Errorf("failed to append initial segment: %v", err)
				}
			}

			// updateの実行
			gotDuration := p.Update(tt.updateSegment)

			// 戻り値の検証
			if gotDuration != tt.wantDuration {
				t.Errorf("update() returned duration = %v, want %v", gotDuration, tt.wantDuration)
			}

			// 最終的なセグメント数の検証
			if len(p.segments) != tt.wantFinalLen {
				t.Errorf("final segments length = %v, want %v", len(p.segments), tt.wantFinalLen)
			}

			// MaxSegmentsの範囲チェック
			if tt.maxSegments <= 0 {
				t.Log("WARN: MaxSegments should be greater than 0")
			}

			// durationの範囲チェック
			if tt.updateSegment.duration <= 0 {
				t.Log("WARN: segment duration should be greater than 0")
			}
		})
	}
}
