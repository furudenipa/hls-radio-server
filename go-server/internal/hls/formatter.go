package hls

import (
	"fmt"
	"strings"
)

// PlaylistContent represents the serialized form of a playlist
type PlaylistContent interface {
	Bytes() []byte
	String() string
}

// PlaylistFormatter defines playlist serialization operations
type PlaylistFormatter interface {
	Format(p *playlist) (PlaylistContent, error)
	Parse(content PlaylistContent) (*playlist, error)
}

// DefaultPlaylistFormatter implements PlaylistFormatter
type DefaultPlaylistFormatter struct{}

func (f *DefaultPlaylistFormatter) Format(p *playlist) (PlaylistContent, error) {
	var lines []string

	p.rwmu.RLock()
	defer p.rwmu.RUnlock()

	// Add header lines
	lines = append(lines, "#EXTM3U")
	lines = append(lines, fmt.Sprintf("#EXT-X-VERSION:%d", p.metadata.version))
	lines = append(lines, fmt.Sprintf("#EXT-X-TARGETDURATION:%.3f", p.metadata.targetDuration))
	lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d", p.metadata.mediaSequence))
	if p.metadata.discontinuitySequence > 0 {
		lines = append(lines, fmt.Sprintf("#EXT-X-DISCONTINUITY-SEQUENCE:%d", p.metadata.discontinuitySequence))
	}

	for _, seg := range p.segments {
		// Add segment lines
		if seg.discontinuity {
			lines = append(lines, "#EXT-X-DISCONTINUITY")
		}
		if seg.duration > 0.0 {
			lines = append(lines, fmt.Sprintf("#EXTINF:%.3f,", seg.duration))
		}
		if seg.uri != "" {
			lines = append(lines, seg.uri)
		}
	}
	return &DefaultPlaylistContent{
		data: []byte(strings.Join(lines, "\n") + "\n"),
	}, nil
}

func (f *DefaultPlaylistFormatter) Parse(content PlaylistContent) (*playlist, error) {
	lines := splitM3U8Lines(content.String())

	p := &playlist{
		metadata: playlistMetadata{
			version: 3, // default version
		},
	}

	parsingHeader := true
	var currentSegment segment

	for _, line := range lines {
		l := m3u8Line(line)

		if parsingHeader {
			if l.hasTag(TagEXTINF) || l.hasTag(TagDISCON) || l.isTS() {
				parsingHeader = false
			} else { // parse header tags
				if l.hasTag(TagVERSION) {
					p.metadata.version = int(l.getTagFloat(TagVERSION))
				} else if l.hasTag(TagMEDIASEQ) {
					p.metadata.mediaSequence = int(l.getTagFloat(TagMEDIASEQ))
				} else if l.hasTag(TagDISCONSEQ) {
					p.metadata.discontinuitySequence = int(l.getTagFloat(TagDISCONSEQ))
				} else if l.hasTag(TagTARGETDURATION) {
					p.metadata.targetDuration = l.getTagFloat(TagTARGETDURATION)
				}
				continue
			}
		}

		// Handle segments
		if !parsingHeader {
			switch {
			case l.hasTag(TagDISCON):
				// Start a new segment with DISCONTINUITY
				currentSegment.discontinuity = true
			case l.hasTag(TagEXTINF):
				// If we have a complete segment, add it
				currentSegment.duration = l.getTagFloat(TagEXTINF)
			case l.isTS():
				// Complete the segment with the TS file
				currentSegment.uri = string(l)
				p.segments = append(p.segments, currentSegment)
				currentSegment = segment{}
			}
		}
	}

	if currentSegment != (segment{}) {
		p.segments = append(p.segments, currentSegment)
	}

	return p, nil
}

// DefaultPlaylistContent implements PlaylistContent
type DefaultPlaylistContent struct {
	data []byte
}

func (c *DefaultPlaylistContent) Bytes() []byte {
	return c.data
}

func (c *DefaultPlaylistContent) String() string {
	return string(c.data)
}
