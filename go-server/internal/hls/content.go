package hls

import (
	"log/slog"
	"path/filepath"
	"strconv"
)

// Content defines the interface for content operations
type Content interface {
	ToSegments() []segment
	ToStreamFilePath(baseDir string) string
	SourcePath() string
	UrlPath() string
	SegmentLocalToGlobal(segment) segment
}

type content struct {
	id          int
	contentType ContentType
	isTmp       bool
	length      int // seconde
	formatter   contentFormatter
}

type contentFormatter interface {
	sourcePath(content) string
	urlPath(content) string
	segmentLocalToGlobal(segment, content) segment
	// parser(string) content
}

type DefaultContentFormatter struct{}

type ContentType string

const (
	audio ContentType = "music"
	news  ContentType = "voice"
)

func NewAudioContent(id int, length int, formatter contentFormatter) *content {
	return &content{
		id:          id,
		contentType: audio,
		isTmp:       false,
		length:      length,
		formatter:   formatter,
	}
}

// TODO: abstract this method
func (d DefaultContentFormatter) sourcePath(c content) string {
	return "/srv/radio/contents/" + string(c.contentType) + "/" + strconv.Itoa(c.id) + "/" + strconv.Itoa(c.id) + ".m3u8"
}

// TODO: abstract this method
func (d DefaultContentFormatter) urlPath(c content) string {
	return "/contents/" + string(c.contentType) + "/" + strconv.Itoa(c.id) + "/" + strconv.Itoa(c.id) + ".m3u8"
}

// TODO: abstract this method
func (d DefaultContentFormatter) segmentLocalToGlobal(seg segment, c content) segment {
	seg.uri = "/contents/" + string(c.contentType) + "/" + strconv.Itoa(c.id) + "/" + seg.uri
	return seg
}

func (c content) SourcePath() string {
	return c.formatter.sourcePath(c)
}

func (c content) UrlPath() string {
	return c.formatter.urlPath(c)
}

func (c content) SegmentLocalToGlobal(seg segment) segment {
	return c.formatter.segmentLocalToGlobal(seg, c)
}

func (c content) ToStreamFilePath(baseDir string) string {
	return filepath.Join(baseDir, "contents", string(c.contentType), strconv.Itoa(c.id), strconv.Itoa(c.id)+".m3u8")
}

func (c content) ToSegments() []segment {
	sourcePath := c.SourcePath()

	// TODO: abstract this using interface
	fs := DefaultFileSystem{}
	pf := DefaultPlaylistFormatter{}

	bytes, err := fs.ReadFile(sourcePath)
	if err != nil {
		slog.Error("failed to read file", "path", sourcePath, "error", err)
		return []segment{}
	}

	pContent := DefaultPlaylistContent{data: bytes}
	playlist, err := pf.Parse(&pContent)
	if err != nil {
		slog.Error("failed to parse playlist", "error", err)
		return []segment{}
	}

	segments := make([]segment, len(playlist.segments))
	for i, seg := range playlist.segments {
		segments[i] = c.SegmentLocalToGlobal(seg)
	}
	return segments
}
