package hls

type Status int

const (
	StatusDefault   Status = iota // 初期状態
	StatusStreaming               // ストリーミング中
	StatusKilled                  // 終了状態
	StatusPaused                  // 一時停止中

)

func (s Status) String() string {
	switch s {
	case StatusDefault:
		return "Default"
	case StatusStreaming:
		return "Streaming"
	case StatusKilled:
		return "Killed"
	case StatusPaused:
		return "Paused"
	default:
		return "Unknown"
	}
}
