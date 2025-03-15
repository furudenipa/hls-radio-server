package hls

import (
	"sync"
)

// PlaylistConfig defines configuration parameters for m3u8 playlist
type PlaylistConfig struct {
	MaxSegments    int
	TargetDuration float64
}

// Playlist represents an m3u8 playlist
type playlist struct {
	metadata playlistMetadata
	segments []segment
	config   PlaylistConfig

	rwmu sync.RWMutex
}

// PlaylistMetadata contains all header information for m3u8 playlist
type playlistMetadata struct {
	version               int
	targetDuration        float64
	mediaSequence         int
	discontinuitySequence int
}

func NewPlaylist(config PlaylistConfig) *playlist {
	return &playlist{
		metadata: playlistMetadata{
			version:               3,
			targetDuration:        config.TargetDuration,
			mediaSequence:         0,
			discontinuitySequence: 0,
		},
		config: config,
	}
}

func (p *playlist) appendSegment(seg segment) error {
	if len(p.segments) >= p.config.MaxSegments {
		return &ErrPlaylistFull{MaxSegments: p.config.MaxSegments}
	}
	if seg.duration <= 0 {
		return &ErrInvalidDuration{Duration: seg.duration}
	}
	p.segments = append(p.segments, seg)
	return nil
}

func (p *playlist) removeOldestSegment() error {
	if len(p.segments) == 0 {
		return &ErrEmptyPlaylist{}
	}
	seg := p.segments[0]
	p.segments = p.segments[1:]

	// update metadata
	p.metadata.mediaSequence += 1
	if seg.discontinuity {
		p.metadata.discontinuitySequence += 1
	}
	return nil
}

// Update adds a segment to the playlist and returns the duration of the oldest segment if the playlist is full
func (p *playlist) Update(seg segment) float64 {
	p.rwmu.Lock()
	defer p.rwmu.Unlock()
	for len(p.segments) >= p.config.MaxSegments {
		if err := p.removeOldestSegment(); err != nil {
			return 0.0
		}
	}

	if err := p.appendSegment(seg); err != nil {
		return 0.0
	}
	if oldestSegment := p.segments[0]; len(p.segments) == p.config.MaxSegments {
		return oldestSegment.duration
	}
	return 0.0
}
