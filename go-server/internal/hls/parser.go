package hls

import (
	"log/slog"
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
