package hls

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"log/slog"
)

const sleepTime = 10

type dj struct {
	manager StreamManager
	logic   logic
}

func (d *dj) Start() {
	go d.manager.Run()

	for {
		content, err := d.logic.Choice()
		if err != nil {
			slog.Error("failed to choose content", "error", err)
			return
		}

		for {
			err = d.manager.Add(content)

			if err == nil {
				// 追加成功したら次のコンテンツを選ぶ
				slog.Info("added content", "content_id", content.id)
				break
			}
			if errors.Is(err, ErrBufferFull) {
				slog.Info("buffer is full, retrying", "content_id", content.id)
				// バッファが一杯なら待機して再試行（同じコンテンツを使用）
				time.Sleep(time.Duration(sleepTime) * time.Second)
				slog.Info("wakup", "content_id", content.id)
				continue
			}
			// その他のエラーは即座に終了
			slog.Error("failed to add content", "error", err)
			return
		}
	}
}

type logic interface {
	Choice() (content, error)
}

type randomLogic struct {
	contents []content
}

func (rl randomLogic) Choice() (content, error) {
	if len(rl.contents) == 0 {
		return content{}, fmt.Errorf("contents is empty")
	}

	return rl.contents[rand.Intn(len(rl.contents))], nil
}
