package hls

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type playlist struct {
	header   []m3u8Line
	segments []m3u8Line

	mediaSeqIndex  int // #EXT-X-MEDIA-SEQUENCE:のindex
	disconSeqIndex int // #EXT-X-DISCONTINUITY-SEQUENCE:のindex

	tsCount int // should remove(?)
	wait    float64
}

type m3u8Line string

type Tag string

const (
	TagEXTINF         Tag = "#EXTINF:"
	TagTARGETDURATION Tag = "#EXT-X-TARGETDURATION:"
	TagMEDIASEQ       Tag = "#EXT-X-MEDIA-SEQUENCE:"
	TagDISCONSEQ      Tag = "#EXT-X-DISCONTINUITY-SEQUENCE:"
	TagDISCON         Tag = "#EXT-X-DISCONTINUITY"
)

func LoadM3U8(filepath string) (*playlist, error) {
	rawText, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("faild to read music m3u8: %w", err)
	}

	lines := splitM3U8Lines(string(rawText))
	header, segments, mediaSeqIndex, disconSeqIndex, tsCount := parseM3U8Lines(lines, filepath)

	return &playlist{
		header:         header,
		segments:       segments,
		mediaSeqIndex:  mediaSeqIndex,
		disconSeqIndex: disconSeqIndex,
		tsCount:        tsCount,
		wait:           -1.0,
	}, nil
}

func NewM3U8(filepath string) (*playlist, error) {
	header := []m3u8Line{
		m3u8Line("#EXTM3U"),
		m3u8Line("#EXT-X-VERSION:3"),
		m3u8Line("#EXT-X-TARGETDURATION:10"),
		m3u8Line("#EXT-X-MEDIA-SEQUENCE:0"),
	}
	var segments []m3u8Line

	p := &playlist{
		header:         header,
		segments:       segments,
		mediaSeqIndex:  3,
		disconSeqIndex: -1,

		tsCount: 0,
		wait:    0.0,
	}

	if err := p.write(filepath); err != nil {
		return p, err
	}
	return p, nil
}

func (p *playlist) write(filepath string) error {
	headerStr := strings.Join(convertM3U8LineSlice(p.header), "\n")
	segmentsStr := strings.Join(convertM3U8LineSlice(p.segments), "\n")
	err := os.WriteFile(filepath, []byte(headerStr+"\n"+segmentsStr+"\n"), 0644)
	if err != nil {
		return fmt.Errorf("ファイル書き込みエラー: %w", err)
	}
	return nil
}

func (p *playlist) SyncFromSource(
	sourcePath, streamFilePath string, stopChan <-chan struct{},
) error {
	source, err := LoadM3U8(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source from %s: %w", sourcePath, err)
	}

	segIndex := 0

	for {
		select {
		case <-stopChan:
			return nil
		// update every m.wait seconds
		case <-time.After(time.Duration(p.wait * float64(time.Second))):
			p.wait = p.update(source, segIndex)
			if err := p.write(streamFilePath); err != nil {
				return nil
			}
			segIndex += 2
			if segIndex >= len(source.segments)-1 {
				// 全セグメント追加終了でreturn
				return nil
			}
		}
	}
}

// This method updates the current `m3u8` instance (`m`)
// using data from the `source` `m3u8` structure.
// Specifically, it utilizes the segment data from the `source` structure,
// starting at the `index`-th line and including the next two lines,
// to perform the update.
func (p *playlist) update(source *playlist, index int) float64 {
	const (
		maxTsCount      = 6
		segmentPairSize = 2 // #EXTINF + *.ts
	)

	linesToRemove := 0
	if p.tsCount >= maxTsCount {
		linesToRemove = segmentPairSize
		p.tsCount -= 1
		p.incrementMEDIASEQ()

		// If the first line is a discontinuity Tag, remove one more line
		if p.segments[0].hasTag(TagDISCON) {
			p.incrementDISCONSEQ()
			linesToRemove = segmentPairSize + 1
		}
	}

	// append #EXT-X-DISCONTINUITY Tag
	if index == 0 {
		p.segments = append(p.segments, m3u8Line(TagDISCON))
	}

	p.segments = append(p.segments, source.segments[index:index+segmentPairSize]...)
	p.segments = p.segments[linesToRemove:]
	p.tsCount += 1

	if linesToRemove == 0 {
		return 0.0
	}

	if l := p.segments[0]; l.hasTag(TagEXTINF) {
		return l.getTagFloat(TagEXTINF)
	}

	// if line0 is TagDISCON, line1 is TagEXTINF
	if l := p.segments[1]; l.hasTag(TagEXTINF) {
		return l.getTagFloat(TagEXTINF)
	}

	return 0.0
}

func (p *playlist) incrementMEDIASEQ() {
	p.header[p.mediaSeqIndex].increment()
}

func (p *playlist) incrementDISCONSEQ() {
	if p.disconSeqIndex != -1 {
		p.header[p.disconSeqIndex].increment()
	} else {
		p.disconSeqIndex = len(p.header)
		p.header = append(p.header, m3u8Line(fmt.Sprintf("%s%d", string(TagDISCONSEQ), 1)))
	}
}

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

// func (l m3u8Line) getTagInt(tag Tag) int {
// if l.hasTag(tag) {
// parsed := string(l)[len(string(tag)):]
// parsed = strings.TrimSuffix(parsed, ",")
// value, err := strconv.Atoi(parsed)
// if err != nil {
// slog.Error("failed to parse line", "tag", string(tag), "line", string(l), "error", err)
// return 0
// }
// return value
// }
// slog.Error("This m3u8Line is not %s: %s", string(tag), string(l))
// return 0
// }

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
