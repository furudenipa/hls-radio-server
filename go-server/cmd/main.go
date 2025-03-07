package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	hls "github.com/furudenipa/hls-radio-server/go-server/internal/hls"
)

func main() {
	p := hls.NewPlaylist(
		hls.PlaylistConfig{
			MaxSegments:    6,
			TargetDuration: 10.0,
		},
	)

	manager := hls.NewPlaylistManager(p)
	dj := hls.NewClassicDJ(manager)
	go dj.Start()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopChan
		manager.Kill()
		os.Exit(0)
	}()

	pFormatter := hls.DefaultPlaylistFormatter{}

	// 例: 動作確認用の簡単なエンドポイント
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	http.HandleFunc("/playlist", func(w http.ResponseWriter, r *http.Request) {
		c, err := pFormatter.Format(p)
		if err != nil {
			http.Error(w, "Failed to format playlist", http.StatusInternalServerError)
			return
		}
		w.Write(c.Bytes())
	})

	fmt.Println("Go server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
