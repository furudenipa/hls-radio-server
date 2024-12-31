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

type m3u8 struct {
	header   []m3u8Line
	segments []m3u8Line

	mediaSeqIndex  int
	disconSeqIndex int

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

func LoadM3U8(filepath string) (*m3u8, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("faild to read music m3u8: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	parsingHeader := true

	var header []m3u8Line
	var segments []m3u8Line
	var tsCount int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		l := m3u8Line(trimmed)

		if parsingHeader {
			if l.HasTag(TagEXTINF) || l.HasTag(TagDISCON) || l.IsTS() {
				parsingHeader = false
				if l.IsTS() {
					segments = append(segments, l.rewriteTsPath(filepath))
				} else {
					segments = append(segments, l)
				}

			} else {
				header = append(header, l)
			}
		} else {
			if l.IsTS() {
				segments = append(segments, l.rewriteTsPath(filepath))
			} else {
				segments = append(segments, l)
			}
		}
	}

	return &m3u8{
		header:         header,
		segments:       segments,
		mediaSeqIndex:  -1,
		disconSeqIndex: -1,
		tsCount:        tsCount,
		wait:           -1.0,
	}, nil
}

func NewM3U8(filepath string) (*m3u8, error) {
	header := []m3u8Line{
		m3u8Line("#EXTM3U"),
		m3u8Line("#EXT-X-VERSION:3"),
		m3u8Line("#EXT-X-TARGETDURATION:10"),
		m3u8Line("#EXT-X-MEDIA-SEQUENCE:0"),
	}
	var segments []m3u8Line

	m := m3u8{
		header:         header,
		segments:       segments,
		mediaSeqIndex:  3,
		disconSeqIndex: -1,

		tsCount: 0,
		wait:    0.0,
	}

	if err := m.WriteToFile(filepath); err != nil {
		return &m, err
	}
	return &m, nil
}

func (m *m3u8) WriteToFile(filepath string) error {
	headerStr := strings.Join(convertM3U8LineSlice(m.header), "\n")
	segmentsStr := strings.Join(convertM3U8LineSlice(m.segments), "\n")
	err := os.WriteFile(filepath, []byte(headerStr+"\n"+segmentsStr+"\n"), 0644)
	if err != nil {
		return fmt.Errorf("ファイル書き込みエラー: %w", err)
	}
	return nil
}

func (m *m3u8) SyncFromSource(sourcePath, streamFilePath string, stopChan <-chan struct{}) error {
	source, err := LoadM3U8(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source from %s: %w", sourcePath, err)
	}

	segIndex := 0

	for {
		select {
		case <-stopChan:
			return nil
		case <-time.After(time.Duration(m.wait * float64(time.Second))):
			m.wait = m.Update(source, segIndex)
			m.WriteToFile(streamFilePath)
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
func (m *m3u8) Update(source *m3u8, index int) float64 {
	const (
		maxTsCount      = 6
		segmentPairSize = 2 // #EXTINF + *.ts
	)

	linesToRemove := 0
	if m.tsCount >= maxTsCount {
		linesToRemove = segmentPairSize
		m.tsCount -= 1
		m.IncrementMEDIASEQ()

		// If the first line is a discontinuity, remove one more line
		if m.segments[0].HasTag(TagDISCON) {
			m.IncrementDISCONSEQ()
			linesToRemove = segmentPairSize + 1
		}
	}

	// append #EXT-X-DISCONTINUITY Tag
	if index == 0 {
		m.segments = append(m.segments, m3u8Line(TagDISCON))
	}

	m.segments = append(m.segments, source.segments[index:index+segmentPairSize]...)
	m.segments = m.segments[linesToRemove:]
	m.tsCount += 1

	if linesToRemove == 0 {
		return 0.0
	}

	if l := m.segments[0]; l.HasTag(TagEXTINF) {
		return l.getTagFloat(TagEXTINF)
	}

	// if line0 is TagDISCON, line1 is TagEXTINF
	if l := m.segments[1]; l.HasTag(TagEXTINF) {
		return l.getTagFloat(TagEXTINF)
	}

	return 0.0
}

func (m *m3u8) IncrementMEDIASEQ() {
	m.header[m.mediaSeqIndex].Increment()
}

func (m *m3u8) IncrementDISCONSEQ() {
	if m.disconSeqIndex != -1 {
		m.header[m.disconSeqIndex].Increment()
	} else {
		m.disconSeqIndex = len(m.header)
		m.header = append(m.header, m3u8Line(fmt.Sprintf("%s%d", string(TagDISCONSEQ), 1)))
	}
}

func (l m3u8Line) HasTag(tag Tag) bool {
	return strings.HasPrefix(string(l), string(tag))
}

func (l m3u8Line) IsTS() bool {
	return strings.HasSuffix(string(l), ".ts")
}

func (l m3u8Line) rewriteTsPath(sourcePath string) m3u8Line {
	return m3u8Line(localToURL(filepath.Join(filepath.Dir(sourcePath), string(l))))
}

func (l m3u8Line) getTagFloat(tag Tag) float64 {
	if l.HasTag(tag) {
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

func (l *m3u8Line) Increment() error {
	sl := string(*l)
	if l.HasTag(TagMEDIASEQ) {
		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagMEDIASEQ)):]))
		if err != nil {
			return fmt.Errorf("this line cannot increment")
		}
		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagMEDIASEQ), n+1))
		return nil
	}

	if l.HasTag(TagDISCONSEQ) {
		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagDISCONSEQ)):]))
		if err != nil {
			return fmt.Errorf("this line cannot increment")
		}
		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagDISCONSEQ), n+1))
		return nil
	}
	return fmt.Errorf("this line cannot increment")
}

func convertM3U8LineSlice(ls []m3u8Line) []string {
	strSlice := make([]string, len(ls))
	for i, s := range ls {
		strSlice[i] = string(s)
	}
	return strSlice
}
