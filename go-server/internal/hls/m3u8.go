package hls

// import (
// 	"fmt"
// 	"log/slog"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"strings"
// )

// // PlaylistConfig defines configuration parameters for m3u8 playlist
// type playlistConfig struct {
// 	MaxSegments    int
// 	TargetDuration float64
// }

// // Playlist represents an m3u8 playlist
// type playlist struct {
// 	metadata playlistMetadata
// 	segments []segment
// 	config   playlistConfig

// 	mediaSeqIndex  int
// 	disconSeqIndex int
// 	wait           float64
// }

// // PlaylistMetadata contains all header information for m3u8 playlist
// type playlistMetadata struct {
// 	version               int
// 	targetDuration        float64
// 	mediaSequence         int
// 	discontinuitySequence int
// }

// // Segment represents a single segment in the playlist
// type segment struct {
// 	duration      float64
// 	uri           string
// 	discontinuity bool
// }

// // PlaylistContent represents the serialized form of a playlist
// type PlaylistContent interface {
// 	Bytes() []byte
// 	String() string
// }

// // PlaylistFormatter defines playlist serialization operations
// type PlaylistFormatter interface {
// 	Format(p *playlist) (PlaylistContent, error)
// 	Parse(content PlaylistContent) (*playlist, error)
// }

// // DefaultPlaylistFormatter implements PlaylistFormatter
// type DefaultPlaylistFormatter struct{}

// func (f *DefaultPlaylistFormatter) Format(p *playlist) (PlaylistContent, error) {
// 	var lines []string

// 	// Add header lines
// 	lines = append(lines, "#EXTM3U")
// 	lines = append(lines, fmt.Sprintf("#EXT-X-VERSION:%d", p.metadata.version))
// 	lines = append(lines, fmt.Sprintf("#EXT-X-TARGETDURATION:%d", p.metadata.targetDuration))
// 	lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d", p.metadata.mediaSequence))
// 	if p.metadata.discontinuitySequence > 0 {
// 		lines = append(lines, fmt.Sprintf("#EXT-X-DISCONTINUITY-SEQUENCE:%d", p.metadata.discontinuitySequence))
// 	}

// 	for _, seg := range p.segments {
// 		// Add segment lines
// 		if seg.discontinuity {
// 			lines = append(lines, "#EXT-X-DISCONTINUITY")
// 		}
// 		if seg.duration > 0.0 {
// 			lines = append(lines, fmt.Sprintf("#EXTINF:%.3f,", seg.duration))
// 		}
// 		if seg.uri != "" {
// 			lines = append(lines, seg.uri)
// 		}
// 	}
// 	return &DefaultPlaylistContent{
// 		data: []byte(strings.Join(lines, "\n") + "\n"),
// 	}, nil
// }

// func (f *DefaultPlaylistFormatter) Parse(content PlaylistContent) (*playlist, error) {
// 	lines := splitM3U8Lines(content.String())

// 	p := &playlist{
// 		metadata: playlistMetadata{
// 			version: 3, // default version
// 		},
// 	}

// 	parsingHeader := true
// 	var currentSegment segment

// 	for _, line := range lines {
// 		l := m3u8Line(line)

// 		if parsingHeader {
// 			if l.hasTag(TagEXTINF) || l.hasTag(TagDISCON) || l.isTS() {
// 				parsingHeader = false
// 			} else { // parse header tags
// 				if l.hasTag(TagVERSION) {
// 					p.metadata.version = int(l.getTagFloat(TagVERSION))
// 				} else if l.hasTag(TagMEDIASEQ) {
// 					p.metadata.mediaSequence = int(l.getTagFloat(TagMEDIASEQ))
// 				} else if l.hasTag(TagDISCONSEQ) {
// 					p.metadata.discontinuitySequence = int(l.getTagFloat(TagDISCONSEQ))
// 				} else if l.hasTag(TagTARGETDURATION) {
// 					p.metadata.targetDuration = l.getTagFloat(TagTARGETDURATION)
// 				}
// 				continue
// 			}
// 		}

// 		// Handle segments
// 		if !parsingHeader {
// 			switch {
// 			case l.hasTag(TagDISCON):
// 				// Start a new segment with DISCONTINUITY
// 				currentSegment.discontinuity = true
// 			case l.hasTag(TagEXTINF):
// 				// If we have a complete segment, add it
// 				currentSegment.duration = l.getTagFloat(TagEXTINF)
// 			case l.isTS():
// 				// Complete the segment with the TS file
// 				currentSegment.uri = string(l)
// 				p.segments = append(p.segments, currentSegment)
// 				currentSegment = segment{}
// 			}
// 		}
// 	}

// 	if currentSegment != (segment{}) {
// 		p.segments = append(p.segments, currentSegment)
// 	}

// 	return p, nil
// }

// // PlaylistStorage defines the storage interface for playlists
// type PlaylistStorage interface {
// 	Store(key string, content PlaylistContent) error
// 	Load(key string) (PlaylistContent, error)
// }

// // FileSystem defines the file system operations
// type FileSystem interface {
// 	ReadFile(path string) ([]byte, error)
// 	WriteFile(path string, data []byte, perm os.FileMode) error
// }

// // DefaultPlaylistContent implements PlaylistContent
// type DefaultPlaylistContent struct {
// 	data []byte
// }

// func (c *DefaultPlaylistContent) Bytes() []byte {
// 	return c.data
// }

// func (c *DefaultPlaylistContent) String() string {
// 	return string(c.data)
// }

// // FileStorage implements PlaylistStorage using the file system
// type FileStorage struct {
// 	fs        FileSystem
// 	directory string
// }

// func NewFileStorage(fs FileSystem, directory string) *FileStorage {
// 	return &FileStorage{
// 		fs:        fs,
// 		directory: directory,
// 	}
// }

// func (s *FileStorage) Store(key string, content PlaylistContent) error {
// 	path := filepath.Join(s.directory, key)
// 	return s.fs.WriteFile(path, content.Bytes(), 0644)
// }

// func (s *FileStorage) Load(key string) (PlaylistContent, error) {
// 	path := filepath.Join(s.directory, key)
// 	data, err := s.fs.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &DefaultPlaylistContent{data: data}, nil
// }

// // DefaultFileSystem implements FileSystem using os package
// type DefaultFileSystem struct{}

// func (fs DefaultFileSystem) ReadFile(path string) ([]byte, error) {
// 	return os.ReadFile(path)
// }

// func (fs DefaultFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
// 	return os.WriteFile(path, data, perm)
// }

// type segmentsQueue struct {
// 	segments      []segment
// 	totalDuration float64
// }

// func (s *segmentsQueue) push(seg segment) {
// 	s.segments = append(s.segments, seg)
// 	s.totalDuration += seg.duration
// }

// func (s *segmentsQueue) pop() (segment, error) {
// 	if len(s.segments) == 0 {
// 		return segment{}, fmt.Errorf("no segments in queue")
// 	}

// 	seg := s.segments[0]
// 	s.segments = s.segments[1:]
// 	s.totalDuration -= seg.duration
// 	return seg, nil
// }

// func (s *segmentsQueue) numSegments() int {
// 	return len(s.segments)
// }

// type m3u8Line string

// type Tag string

// const (
// 	TagVERSION        Tag = "#EXT-X-VERSION:"
// 	TagTARGETDURATION Tag = "#EXT-X-TARGETDURATION:"
// 	TagMEDIASEQ       Tag = "#EXT-X-MEDIA-SEQUENCE:"
// 	TagDISCONSEQ      Tag = "#EXT-X-DISCONTINUITY-SEQUENCE:"
// 	TagEXTINF         Tag = "#EXTINF:"
// 	TagDISCON         Tag = "#EXT-X-DISCONTINUITY"
// )

// // func LoadM3U8(filepath string) (*playlist, error) {}
// // refactor

// func NewPlaylist(config playlistConfig) *playlist {
// 	return &playlist{
// 		metadata: playlistMetadata{
// 			version:               3,
// 			targetDuration:        config.TargetDuration,
// 			mediaSequence:         0,
// 			discontinuitySequence: 0,
// 		},
// 		config: config,
// 	}
// }

// // func (p *playlist) write(filepath string) error {}

// func (p *playlist) appendSegment(seg segment) error {
// 	if len(p.segments) >= p.config.MaxSegments {
// 		return fmt.Errorf("hoge")
// 	}
// 	p.segments = append(p.segments, seg)
// 	return nil
// }

// func (p *playlist) removeOldestSegment() {
// 	if len(p.segments) > 0 {
// 		seg := p.segments[0]
// 		p.segments = p.segments[1:]

// 		// update metadata
// 		p.metadata.mediaSequence += 1
// 		if seg.discontinuity {
// 			p.metadata.discontinuitySequence += 1
// 		}
// 	}
// }

// func (p *playlist) update(seg segment) float64 {
// 	for len(p.segments) < p.config.MaxSegments {
// 		p.removeOldestSegment()
// 	}

// 	p.appendSegment(seg)

// 	if oldestSegment := p.segments[0]; len(p.segments) == p.config.MaxSegments {
// 		return oldestSegment.duration
// 	}
// 	return 0.0
// }

// func (l m3u8Line) hasTag(tag Tag) bool {
// 	return strings.HasPrefix(string(l), string(tag))
// }

// func (l m3u8Line) isTS() bool {
// 	return strings.HasSuffix(string(l), ".ts")
// }

// func (l m3u8Line) rewriteTsPath(sourcePath string) m3u8Line {
// 	// TSファイルの相対パスを絶対パスに変換
// 	tsPath := string(l)
// 	sourceDir := filepath.Dir(sourcePath)

// 	// 相対パスを絶対パスに変換し、パスを正規化
// 	absPath := filepath.Clean(filepath.Join(sourceDir, tsPath))

// 	// パスをスラッシュ形式に統一
// 	normalizedPath := filepath.ToSlash(absPath)

// 	// URLに変換
// 	return m3u8Line(localToURL(normalizedPath))
// }

// func (l m3u8Line) getTagFloat(tag Tag) float64 {
// 	if l.hasTag(tag) {
// 		parsed := string(l)[len(string(tag)):]
// 		parsed = strings.TrimSuffix(parsed, ",")
// 		value, err := strconv.ParseFloat(parsed, 64)
// 		if err != nil {
// 			slog.Error("failed to parse line", "tag", string(tag), "line", string(l), "error", err)
// 			return 0.0
// 		}
// 		return value
// 	}
// 	slog.Error("This m3u8Line is not %s: %s", string(tag), string(l))
// 	return 0.0
// }

// func (l *m3u8Line) increment() error {
// 	sl := string(*l)
// 	switch {
// 	case l.hasTag(TagMEDIASEQ):
// 		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagMEDIASEQ)):]))
// 		if err != nil {
// 			return fmt.Errorf("this line cannot increment")
// 		}
// 		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagMEDIASEQ), n+1))
// 	case l.hasTag(TagDISCONSEQ):
// 		n, err := strconv.Atoi(strings.TrimSpace(sl[len(string(TagDISCONSEQ)):]))
// 		if err != nil {
// 			return fmt.Errorf("this line cannot increment")
// 		}
// 		*l = m3u8Line(fmt.Sprintf("%s%d", string(TagDISCONSEQ), n+1))
// 	default:
// 		return fmt.Errorf("this line cannot increment")
// 	}
// 	return nil
// }

// // 与えられた生文字列を行単位に分割し、空白を除いて返す
// func splitM3U8Lines(rawText string) []string {
// 	rawLines := strings.Split(rawText, "\n")
// 	var lines []string
// 	for _, line := range rawLines {
// 		trimmed := strings.TrimSpace(line)
// 		if trimmed != "" {
// 			lines = append(lines, trimmed)
// 		}
// 	}
// 	return lines
// }

// func parseM3U8Lines(lines []string, sourcePath string) ([]m3u8Line, []m3u8Line, int, int, int) {
// 	parsingHeader := true
// 	var header []m3u8Line
// 	var segments []m3u8Line
// 	var mediaSeqIndex = -1
// 	var disconSeqIndex = -1
// 	var tsCount int
// 	for i, line := range lines {
// 		l := m3u8Line(line)

// 		if parsingHeader {
// 			if l.hasTag(TagEXTINF) || l.hasTag(TagDISCON) || l.isTS() {
// 				parsingHeader = false
// 			} else {
// 				header = append(header, l)
// 				if l.hasTag(TagMEDIASEQ) {
// 					mediaSeqIndex = i
// 				} else if l.hasTag(TagDISCONSEQ) {
// 					disconSeqIndex = i
// 				}
// 			}
// 		}

// 		if !parsingHeader {
// 			if l.isTS() {
// 				tsCount += 1
// 				segments = append(segments, l.rewriteTsPath(sourcePath))
// 			} else {
// 				segments = append(segments, l)
// 			}
// 		}
// 	}

// 	return header, segments, mediaSeqIndex, disconSeqIndex, tsCount
// }

// func convertM3U8LineSlice(ls []m3u8Line) []string {
// 	strSlice := make([]string, len(ls))
// 	for i, s := range ls {
// 		strSlice[i] = string(s)
// 	}
// 	return strSlice
// }
