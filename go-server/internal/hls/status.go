package hls

type Status int

const (
	StatusDefault   Status = iota // 初期状態
	StatusStreaming               // ストリーミング中
	StatusStopped                 // 停止状態
)

func (s Status) String() string {
	switch s {
	case StatusDefault:
		return "Default"
	case StatusStreaming:
		return "Streaming"
	case StatusStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}
