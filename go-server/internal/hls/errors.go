package hls

import "fmt"

// ErrPlaylistFull はプレイリストが最大セグメント数に達したときのエラー
type ErrPlaylistFull struct {
	MaxSegments int
}

func (e *ErrPlaylistFull) Error() string {
	return fmt.Sprintf("playlist is full: max segments limit (%d) reached", e.MaxSegments)
}

// ErrEmptyPlaylist はプレイリストが空のときの操作に対するエラー
type ErrEmptyPlaylist struct{}

func (e *ErrEmptyPlaylist) Error() string {
	return "operation failed: playlist is empty"
}

// ErrInvalidDuration はセグメントの長さが不正な場合のエラー
type ErrInvalidDuration struct {
	Duration float64
}

func (e *ErrInvalidDuration) Error() string {
	return fmt.Sprintf("invalid segment duration: %f", e.Duration)
}
