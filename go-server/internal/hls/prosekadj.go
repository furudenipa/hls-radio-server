package hls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"strconv"
)

func NewProsekaDJ(m3u8Manager *m3u8Manager) *dj {
	const jsonPath = "/srv/radio/contents/index.json"
	contents := NewProsekaContentsFromJson(jsonPath)
	return &dj{
		manager: m3u8Manager,
		logic: randomLogic{
			contents: contents,
		},
	}

}

// =======================
// == gpt 4o no copy-pe ==
// =======================
type Track struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Length int    `json:"length"`
	Artist string `json:"artist"`
	M3U8   string `json:"m3u8"`
}

func NewProsekaContentsFromJson(filepath string) []content {
	// ファイルを読み込む
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}

	// JSONをパース
	var tracks []Track
	if err := json.Unmarshal(data, &tracks); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil
	}

	// contentリストに変換
	var contents []content
	for _, track := range tracks {
		i, err := strconv.Atoi(track.ID)
		if err != nil {
			slog.Error("trackID cant convert to Int", "err", err)
			continue
		}
		contents = append(contents, content{
			id:          i,
			contentType: audio, // 固定値
			isTmp:       false, // 固定値
			length:      track.Length,
		})
	}

	return contents
}
