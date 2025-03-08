package main

import (
	"fmt"

	hls "github.com/furudenipa/hls-radio-server/go-server/internal/hls"
)

func main() {
	toSegT()
	// p := hls.NewPlaylist(
	// 	hls.PlaylistConfig{
	// 		MaxSegments:    6,
	// 		TargetDuration: 10.0,
	// 	},
	// )
	// formatter := hls.DefaultPlaylistFormatter{}

	// pm := hls.NewPlaylistManager(p)
	// go pm.Run()
	// fmt.Println(pm.Add(*hls.NewAudioContent(1, 10, hls.DefaultContentFormatter{})))
	// time.Sleep(20 * time.Second)
	// c, _ := formatter.Format(p)
	// fmt.Println(c.String())
}

func toSegT() {

	ac := hls.NewAudioContent(1, 10, hls.DefaultContentFormatter{})
	fmt.Println(ac.ToSegments())

}
