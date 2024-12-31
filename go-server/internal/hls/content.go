package hls

import (
	"path/filepath"
	"strconv"
)

type content struct {
	id          int
	contentType ContentType
	isTmp       bool
	length      int // seconde
}

type ContentType string

const (
	audio ContentType = "music"
	news  ContentType = "voice"
)

func NewAudioContent(id int, length int) *content {
	return &content{
		id:          id,
		contentType: audio,
		isTmp:       false,
		length:      length,
	}
}

func (c content) ToStreamFilePath(baseDir string) string {
	return filepath.Join(baseDir, "contents", string(c.contentType), strconv.Itoa(c.id), strconv.Itoa(c.id)+".m3u8")
}
