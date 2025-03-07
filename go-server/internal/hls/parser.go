package hls

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
)

type m3u8Line string

type Tag string

const (
	TagVERSION        Tag = "#EXT-X-VERSION:"
	TagTARGETDURATION Tag = "#EXT-X-TARGETDURATION:"
	TagMEDIASEQ       Tag = "#EXT-X-MEDIA-SEQUENCE:"
	TagDISCONSEQ      Tag = "#EXT-X-DISCONTINUITY-SEQUENCE:"
	TagEXTINF         Tag = "#EXTINF:"
	TagDISCON         Tag = "#EXT-X-DISCONTINUITY"
)

func (l m3u8Line) hasTag(tag Tag) bool {
	return strings.HasPrefix(string(l), string(tag))
}

func (l m3u8Line) isTS() bool {
	return strings.HasSuffix(string(l), ".ts")
}

func (l m3u8Line) rewriteTsPath(sourcePath string) m3u8Line {
	// TSファイルの相対パスを絶対パスに変換
	tsPath := string(l)
	sourceDir := filepath.Dir(sourcePath)

	// 相対パスを絶対パスに変換し、パスを正規化
	absPath := filepath.Clean(filepath.Join(sourceDir, tsPath))

	// パスをスラッシュ形式に統一
	normalizedPath := filepath.ToSlash(absPath)

	// URLに変換
	return m3u8Line(localToURL(normalizedPath))
}

func (l m3u8Line) getTagFloat(tag Tag) float64 {
	if l.hasTag(tag) {
		parsed := string(l)[len(string(tag)):]
		parsed = strings.TrimSuffix(parsed, ",")
		value, err := strconv.ParseFloat(parsed, 64)
		if err != nil {
			slog.Error("failed to parse line", "tag", string(tag), "line", string(l), "error", err)
			return 0.0
		}
		return value
	}
	slog.Error("This m3u8Line is not %s: %s", string(tag), string(l))
	return 0.0
}

func (l *m3u8Line) increment() error {
	sl := string(*l)
	switch {
	case l.hasTag(TagMEDIASEQ):
		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagMEDIASEQ)):]))
		if err != nil {
			return fmt.Errorf("this line cannot increment")
		}
		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagMEDIASEQ), n+1))
	case l.hasTag(TagDISCONSEQ):
		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagDISCONSEQ)):]))
		if err != nil {
			return fmt.Errorf("this line cannot increment")
		}
		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagDISCONSEQ), n+1))
	default:
		return fmt.Errorf("this line cannot increment")
	}
	return nil
}

// 与えられた生文字列を行単位に分割し、空白を除いて返す
func splitM3U8Lines(rawText string) []string {
	rawLines := strings.Split(rawText, "\n")
	var lines []string
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func parseM3U8Lines(lines []string, sourcePath string) ([]m3u8Line, []m3u8Line, int, int, int) {
	parsingHeader := true
	var header []m3u8Line
	var segments []m3u8Line
	var mediaSeqIndex = -1
	var disconSeqIndex = -1
	var tsCount int
	for i, line := range lines {
		l := m3u8Line(line)

		if parsingHeader {
			if l.hasTag(TagEXTINF) || l.hasTag(TagDISCON) || l.isTS() {
				parsingHeader = false
			} else {
				header = append(header, l)
				if l.hasTag(TagMEDIASEQ) {
					mediaSeqIndex = i
				} else if l.hasTag(TagDISCONSEQ) {
					disconSeqIndex = i
				}
			}
		}

		if !parsingHeader {
			if l.isTS() {
				tsCount += 1
				segments = append(segments, l.rewriteTsPath(sourcePath))
			} else {
				segments = append(segments, l)
			}
		}
	}

	return header, segments, mediaSeqIndex, disconSeqIndex, tsCount
}

func convertM3U8LineSlice(ls []m3u8Line) []string {
	strSlice := make([]string, len(ls))
	for i, s := range ls {
		strSlice[i] = string(s)
	}
	return strSlice
}
