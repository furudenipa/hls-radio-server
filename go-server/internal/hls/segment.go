package hls

import (
	"fmt"
)

// segment represents a single segment in the playlist
type segment struct {
	duration      float64
	uri           string
	discontinuity bool
}

type segmentsQueue struct {
	segments      []segment
	totalDuration float64
}

func NewSegment(duration float64, uri string, discontinuity bool) segment {
	return segment{
		duration:      duration,
		uri:           uri,
		discontinuity: discontinuity,
	}
}

func (s *segment) String() string {
	return fmt.Sprintf("duration: %.3f, uri: %s, discontinuity: %t", s.duration, s.uri, s.discontinuity)
}

func (s *segmentsQueue) push(seg segment) {
	s.segments = append(s.segments, seg)
	s.totalDuration += seg.duration
	fmt.Println("pushed segment", seg)
}

func (s *segmentsQueue) pop() (segment, error) {
	if len(s.segments) == 0 {
		return segment{}, fmt.Errorf("no segments in queue")
	}

	seg := s.segments[0]
	s.segments = s.segments[1:]
	s.totalDuration -= seg.duration
	return seg, nil
}
