package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"web-server-project/internal/hls"
)

func main() {
	mux := http.NewServeMux()

	manager := hls.NewStreamM3U8Manager("/srv/radio/", "/srv/radio/stations/ProsekaStation/stream.m3u8")
	dj := hls.NewProsekaDJ(manager)
	go dj.Start()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopChan
		manager.Stop()
		os.Exit(0)
	}()
	// 例: 動作確認用の簡単なエンドポイント
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	fmt.Println("Go server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
