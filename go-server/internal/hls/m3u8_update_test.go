package hls

import (
	"reflect"
	"testing"
)

func TestPlaylist_Update(t *testing.T) {
	// ソースプレイリストの作成（十分な量のセグメントを含む）
	source := &playlist{
		header: []m3u8Line{
			"#EXTM3U",
			"#EXT-X-VERSION:3",
			"#EXT-X-TARGETDURATION:10",
			"#EXT-X-MEDIA-SEQUENCE:0",
		},
		segments: []m3u8Line{
			"#EXTINF:9.009,",
			"segment1.ts",
			"#EXTINF:9.009,",
			"segment2.ts",
			"#EXTINF:9.009,",
			"segment3.ts",
			"#EXTINF:9.009,",
			"segment4.ts",
			"#EXTINF:9.009,",
			"segment5.ts",
			"#EXTINF:9.009,",
			"segment6.ts",
			"#EXTINF:9.009,",
			"segment7.ts",
			"#EXTINF:9.009,",
			"segment8.ts",
		},
		mediaSeqIndex:  3,
		disconSeqIndex: -1,
		tsCount:        8,
	}

	tests := []struct {
		name           string
		targetPlaylist *playlist
		sourceIndex    int
		wantTsCount    int
		wantWait       float64
		wantSegments   []m3u8Line // 期待されるセグメント
	}{
		{
			name: "初期セグメント追加",
			targetPlaylist: &playlist{
				header: []m3u8Line{
					"#EXTM3U",
					"#EXT-X-VERSION:3",
					"#EXT-X-TARGETDURATION:10",
					"#EXT-X-MEDIA-SEQUENCE:0",
				},
				segments:      []m3u8Line{},
				mediaSeqIndex: 3,
				tsCount:       0,
			},
			sourceIndex: 0,
			wantTsCount: 1,
			wantWait:    0.0,
			wantSegments: []m3u8Line{
				"#EXT-X-DISCONTINUITY",
				"#EXTINF:9.009,",
				"segment1.ts",
			},
		},
		{
			name: "2番目のセグメントペア追加",
			targetPlaylist: &playlist{
				header: []m3u8Line{
					"#EXTM3U",
					"#EXT-X-VERSION:3",
					"#EXT-X-TARGETDURATION:10",
					"#EXT-X-MEDIA-SEQUENCE:0",
				},
				segments: []m3u8Line{
					"#EXT-X-DISCONTINUITY",
					"#EXTINF:9.009,",
					"segment1.ts",
				},
				mediaSeqIndex: 3,
				tsCount:       1,
			},
			sourceIndex: 2,
			wantTsCount: 2,
			wantWait:    0.0,
			wantSegments: []m3u8Line{
				"#EXT-X-DISCONTINUITY",
				"#EXTINF:9.009,",
				"segment1.ts",
				"#EXTINF:9.009,",
				"segment2.ts",
			},
		},
		{
			name: "最大セグメント数到達時の更新",
			targetPlaylist: &playlist{
				header: []m3u8Line{
					"#EXTM3U",
					"#EXT-X-VERSION:3",
					"#EXT-X-TARGETDURATION:10",
					"#EXT-X-MEDIA-SEQUENCE:0",
				},
				segments: []m3u8Line{
					"#EXT-X-DISCONTINUITY",
					"#EXTINF:9.009,",
					"segment1.ts",
					"#EXTINF:9.009,",
					"segment2.ts",
					"#EXTINF:9.009,",
					"segment3.ts",
					"#EXTINF:9.009,",
					"segment4.ts",
					"#EXTINF:9.009,",
					"segment5.ts",
					"#EXTINF:9.009,",
					"segment6.ts",
				},
				mediaSeqIndex: 3,
				tsCount:       6,
			},
			sourceIndex: 12, // 6番目のセグメントペアの開始インデックス
			wantTsCount: 6,
			wantWait:    9.009,
			wantSegments: []m3u8Line{
				"#EXTINF:9.009,",
				"segment2.ts",
				"#EXTINF:9.009,",
				"segment3.ts",
				"#EXTINF:9.009,",
				"segment4.ts",
				"#EXTINF:9.009,",
				"segment5.ts",
				"#EXTINF:9.009,",
				"segment6.ts",
				"#EXTINF:9.009,",
				"segment7.ts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait := tt.targetPlaylist.update(source, tt.sourceIndex)

			// tsCountの検証
			if tt.targetPlaylist.tsCount != tt.wantTsCount {
				t.Errorf("update() tsCount = %v, want %v", tt.targetPlaylist.tsCount, tt.wantTsCount)
			}

			// wait値の検証
			if wait != tt.wantWait {
				t.Errorf("update() wait = %v, want %v", wait, tt.wantWait)
			}

			// 追加されたセグメントの検証
			if !reflect.DeepEqual(tt.targetPlaylist.segments, tt.wantSegments) {
				t.Errorf("update() segments = %v, want %v", tt.targetPlaylist.segments, tt.wantSegments)
			}
		})
	}
}

func TestSequenceIncrement(t *testing.T) {
	tests := []struct {
		name               string
		initialMediaSeq    string
		initialDisconSeq   string
		wantMediaSeqAfter  string
		wantDisconSeqAfter string
		wantMediaSeqErr    bool
		wantDisconSeqErr   bool
		includeDisconSeq   bool
	}{
		{
			name:              "正常なシーケンス番号の増加",
			initialMediaSeq:   "#EXT-X-MEDIA-SEQUENCE:0",
			wantMediaSeqAfter: "#EXT-X-MEDIA-SEQUENCE:1",
			wantMediaSeqErr:   false,
			includeDisconSeq:  false,
		},
		{
			name:               "DISCONTINUITYシーケンス番号の増加",
			initialMediaSeq:    "#EXT-X-MEDIA-SEQUENCE:5",
			initialDisconSeq:   "#EXT-X-DISCONTINUITY-SEQUENCE:2",
			wantMediaSeqAfter:  "#EXT-X-MEDIA-SEQUENCE:6",
			wantDisconSeqAfter: "#EXT-X-DISCONTINUITY-SEQUENCE:3",
			wantMediaSeqErr:    false,
			wantDisconSeqErr:   false,
			includeDisconSeq:   true,
		},
		{
			name:              "不正なメディアシーケンス番号",
			initialMediaSeq:   "#EXT-X-MEDIA-SEQUENCE:invalid",
			wantMediaSeqAfter: "#EXT-X-MEDIA-SEQUENCE:invalid",
			wantMediaSeqErr:   true,
			includeDisconSeq:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &playlist{
				header:         []m3u8Line{m3u8Line(tt.initialMediaSeq)},
				mediaSeqIndex:  0,
				disconSeqIndex: -1,
			}

			if tt.includeDisconSeq {
				p.header = append(p.header, m3u8Line(tt.initialDisconSeq))
				p.disconSeqIndex = 1
			}

			// メディアシーケンス番号の更新テスト
			p.incrementMEDIASEQ()
			if got := string(p.header[p.mediaSeqIndex]); got != tt.wantMediaSeqAfter {
				t.Errorf("incrementMEDIASEQ() got = %v, want %v", got, tt.wantMediaSeqAfter)
			}

			// DISCONTINUITYシーケンス番号の更新テスト（含まれている場合）
			if tt.includeDisconSeq {
				p.incrementDISCONSEQ()
				if got := string(p.header[p.disconSeqIndex]); got != tt.wantDisconSeqAfter {
					t.Errorf("incrementDISCONSEQ() got = %v, want %v", got, tt.wantDisconSeqAfter)
				}
			}
		})
	}
}

func TestGetTagFloat(t *testing.T) {
	tests := []struct {
		name     string
		line     m3u8Line
		tag      Tag
		want     float64
		wantZero bool
	}{
		{
			name:     "正常なEXTINF値",
			line:     "#EXTINF:9.009,",
			tag:      TagEXTINF,
			want:     9.009,
			wantZero: false,
		},
		{
			name:     "無効なEXTINF値",
			line:     "#EXTINF:invalid,",
			tag:      TagEXTINF,
			want:     0.0,
			wantZero: true,
		},
		{
			name:     "異なるタグ",
			line:     "#EXTINF:9.009,",
			tag:      TagDISCON,
			want:     0.0,
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.line.getTagFloat(tt.tag)
			if tt.wantZero {
				if got != 0.0 {
					t.Errorf("getTagFloat() = %v, want 0.0", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("getTagFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
