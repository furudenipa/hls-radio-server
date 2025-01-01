package hls

func NewClassicDJ(m3u8Manager *m3u8Manager) *dj {
	contents := []content{
		*NewAudioContent(1, 85),
		*NewAudioContent(2, 196),
		*NewAudioContent(3, 43),
	}
	return &dj{
		manager: m3u8Manager,
		logic: randomLogic{
			contents: contents,
		},
	}
}
