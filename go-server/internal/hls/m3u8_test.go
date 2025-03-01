package hls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadM3U8(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{
			name:    "基本的なM3U8ファイル",
			file:    "testdata/basic.m3u8",
			wantErr: false,
		},
		{
			name:    "最大セグメント数のM3U8ファイル",
			file:    "testdata/max_segments.m3u8",
			wantErr: false,
		},
		{
			name:    "DISCONTINUITYを含むM3U8ファイル",
			file:    "testdata/discontinuity.m3u8",
			wantErr: false,
		},
		{
			name:    "不正なヘッダーのM3U8ファイル",
			file:    "testdata/invalid_header.m3u8",
			wantErr: false, // ヘッダーが不完全でもエラーにはならない
		},
		{
			name:    "不正なEXTINF値のM3U8ファイル",
			file:    "testdata/invalid_extinf.m3u8",
			wantErr: false, // パース時にはエラーにならない（使用時にエラー）
		},
		{
			name:    "不正なシーケンス番号のM3U8ファイル",
			file:    "testdata/invalid_sequence.m3u8",
			wantErr: false, // パース時にはエラーにならない（increment時にエラー）
		},
		{
			name:    "空のM3U8ファイル",
			file:    "testdata/empty.m3u8",
			wantErr: false,
		},
		{
			name:    "特殊文字を含むパスのM3U8ファイル",
			file:    "testdata/special_chars.m3u8",
			wantErr: false,
		},
		{
			name:    "存在しないファイル",
			file:    "testdata/nonexistent.m3u8",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadM3U8(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadM3U8() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewM3U8(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "m3u8test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "新規M3U8ファイル作成",
			path:    filepath.Join(tmpDir, "new.m3u8"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewM3U8(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewM3U8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if p == nil {
				t.Error("NewM3U8() returned nil playlist")
				return
			}

			// 基本的なプロパティの検証
			if len(p.header) != 4 {
				t.Errorf("NewM3U8() header length = %v, want 4", len(p.header))
			}
			if p.mediaSeqIndex != 3 {
				t.Errorf("NewM3U8() mediaSeqIndex = %v, want 3", p.mediaSeqIndex)
			}
			if p.disconSeqIndex != -1 {
				t.Errorf("NewM3U8() disconSeqIndex = %v, want -1", p.disconSeqIndex)
			}
		})
	}
}

func TestPlaylist_Write(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "m3u8test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		playlist *playlist
		wantErr  bool
	}{
		{
			name: "基本的なプレイリストの書き込み",
			playlist: &playlist{
				header: []m3u8Line{
					"#EXTM3U",
					"#EXT-X-VERSION:3",
					"#EXT-X-TARGETDURATION:10",
					"#EXT-X-MEDIA-SEQUENCE:0",
				},
				segments: []m3u8Line{
					"#EXTINF:9.009,",
					"segment1.ts",
				},
				mediaSeqIndex:  3,
				disconSeqIndex: -1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, "test.m3u8")
			if err := tt.playlist.write(path); (err != nil) != tt.wantErr {
				t.Errorf("playlist.write() error = %v, wantErr %v", err, tt.wantErr)
			}

			// ファイルが作成されたことを確認
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Error("playlist.write() did not create file")
			}
		})
	}
}

func TestLineOperations(t *testing.T) {
	tests := []struct {
		name string
		line m3u8Line
		tag  Tag
		want bool
	}{
		{
			name: "EXTINFタグの検出",
			line: "#EXTINF:9.009,",
			tag:  TagEXTINF,
			want: true,
		},
		{
			name: "DISCONTINUITYタグの検出",
			line: "#EXT-X-DISCONTINUITY",
			tag:  TagDISCON,
			want: true,
		},
		{
			name: "存在しないタグ",
			line: "#EXTINF:9.009,",
			tag:  TagDISCON,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.line.hasTag(tt.tag); got != tt.want {
				t.Errorf("hasTag() = %v, want %v", got, tt.want)
			}
		})
	}

	// TSファイルの検出テスト
	tsTests := []struct {
		name string
		line m3u8Line
		want bool
	}{
		{
			name: "通常のTSファイル",
			line: "segment1.ts",
			want: true,
		},
		{
			name: "特殊文字を含むパスのTSファイル",
			line: "special path/segment1.ts",
			want: true,
		},
		{
			name: "TSファイルではない",
			line: "#EXTINF:9.009,",
			want: false,
		},
	}

	for _, tt := range tsTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.line.isTS(); got != tt.want {
				t.Errorf("isTS() = %v, want %v", got, tt.want)
			}
		})
	}
}
