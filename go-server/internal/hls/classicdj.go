package hls

func NewClassicDJ(m3u8Manager *M3U8Manager) *DJ {
	contents := []content{
		*NewAudioContent(1, 85),
		*NewAudioContent(2, 196),
		*NewAudioContent(3, 43),
	}
	return &DJ{
		m3u8Manager: m3u8Manager,
		L: RandomLogic{
			contents: contents,
		},
	}
}
