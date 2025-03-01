package hls

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConvertM3U8LineSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []m3u8Line
		want  []string
	}{
		{
			name:  "空のスライス",
			input: []m3u8Line{},
			want:  []string{},
		},
		{
			name: "1要素のスライス",
			input: []m3u8Line{
				"#EXTM3U",
			},
			want: []string{
				"#EXTM3U",
			},
		},
		{
			name: "基本的なM3U8行",
			input: []m3u8Line{
				"#EXTM3U",
				"#EXT-X-VERSION:3",
				"#EXTINF:9.009,",
				"segment1.ts",
			},
			want: []string{
				"#EXTM3U",
				"#EXT-X-VERSION:3",
				"#EXTINF:9.009,",
				"segment1.ts",
			},
		},
		{
			name: "特殊文字を含む行",
			input: []m3u8Line{
				"#EXTM3U",
				"special path/segment1.ts",
				"path/with#hash/segment2.ts",
				"日本語パス/セグメント.ts",
			},
			want: []string{
				"#EXTM3U",
				"special path/segment1.ts",
				"path/with#hash/segment2.ts",
				"日本語パス/セグメント.ts",
			},
		},
		{
			name:  "大きなスライス",
			input: generateLargem3u8LineSlice(100),
			want:  generateLargeStringSlice(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertM3U8LineSlice(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertM3U8LineSlice() = %v, want %v", got, tt.want)
			}

			// スライスの容量がぴったりであることを確認
			if cap(got) != len(got) {
				t.Errorf("convertM3U8LineSlice() capacity = %v, want %v", cap(got), len(got))
			}
		})
	}
}

func TestRewriteTsPath(t *testing.T) {
	// テスト用のベースディレクトリ
	baseDir := "/srv/radio"

	tests := []struct {
		name       string
		line       m3u8Line
		sourcePath string
		want       m3u8Line
	}{
		{
			name:       "通常のTSファイル",
			line:       "segment1.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/segment1.ts",
		},
		{
			name:       "サブディレクトリのTSファイル",
			line:       "segments/segment1.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/segments/segment1.ts",
		},
		{
			name:       "特殊文字を含むパス",
			line:       "special path/segment1.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/special path/segment1.ts",
		},
		{
			name:       "親ディレクトリへの相対パス",
			line:       "../segments/segment1.ts",
			sourcePath: filepath.Join(baseDir, "subdir", "playlist.m3u8"),
			want:       "/segments/segment1.ts",
		},
		{
			name:       "同一ディレクトリのTSファイル",
			line:       "./segment1.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/segment1.ts",
		},
		{
			name:       "日本語を含むパス",
			line:       "セグメント/ファイル.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/セグメント/ファイル.ts",
		},
		{
			name:       "URLエンコードが必要な特殊文字",
			line:       "segments/file with #hash.ts",
			sourcePath: filepath.Join(baseDir, "playlist.m3u8"),
			want:       "/segments/file with #hash.ts",
		},
		{
			name:       "/srv/radio外のパス",
			line:       "segment1.ts",
			sourcePath: "/other/path/playlist.m3u8",
			want:       "/other/path/segment1.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.line.rewriteTsPath(tt.sourcePath)
			// パスセパレータを/に統一して比較
			gotNormalized := filepath.ToSlash(string(got))
			if gotNormalized != string(tt.want) {
				t.Errorf("rewriteTsPath() = %v, want %v", gotNormalized, tt.want)
			}
		})
	}
}

// テストヘルパー関数
func generateLargem3u8LineSlice(size int) []m3u8Line {
	result := make([]m3u8Line, size)
	for i := 0; i < size; i++ {
		result[i] = m3u8Line(fmt.Sprintf("line%d", i))
	}
	return result
}

func generateLargeStringSlice(size int) []string {
	result := make([]string, size)
	for i := 0; i < size; i++ {
		result[i] = fmt.Sprintf("line%d", i)
	}
	return result
}
