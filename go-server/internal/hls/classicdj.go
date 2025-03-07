package hls

func NewClassicDJ(pManager *playlistManager) *dj {
	contents := []content{
		*NewAudioContent(1, 85, DefaultContentFormatter{}),
		*NewAudioContent(2, 196, DefaultContentFormatter{}),
		*NewAudioContent(3, 43, DefaultContentFormatter{}),
	}
	return &dj{
		manager: pManager,
		logic: randomLogic{
			contents: contents,
		},
	}
}
