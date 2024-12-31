package hls

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

const sleepTime = 10
const necessaryRemainingTime = 80

type DJ struct {
	m3u8Manager   *M3U8Manager
	L             Logic
	remainingTime int // second
}

func (d *DJ) Start() error {
	go d.m3u8Manager.Run()
	for {
		for {
			if d.remainingTime >= necessaryRemainingTime {
				break
			}
			content, err := d.L.Choice()
			if err != nil {
				return err
			}
			d.remainingTime += content.length
			d.m3u8Manager.Add(content)
		}
		slog.Info("DJ: ", "remainingTime", d.remainingTime)
		time.Sleep(sleepTime * time.Second)
		d.remainingTime -= sleepTime
	}
}

type Logic interface {
	Choice() (content, error)
}

type RandomLogic struct {
	contents []content
}

func (rl RandomLogic) Choice() (content, error) {
	if len(rl.contents) == 0 {
		return content{}, fmt.Errorf("contents is empty")
	}

	return rl.contents[rand.Intn(len(rl.contents))], nil
}
