package hls

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

const sleepTime = 10
const necessaryRemainingTime = 80

type dj struct {
	manager       StreamManager
	logic         logic
	remainingTime int // second
}

func (d *dj) Start() error {
	go d.manager.Run()
	for {
		for {
			if d.remainingTime >= necessaryRemainingTime {
				break
			}
			content, err := d.logic.Choice()
			if err != nil {
				return err
			}
			d.remainingTime += content.length
			d.manager.Add(content)
		}
		slog.Info("DJ: ", "remainingTime", d.remainingTime)
		time.Sleep(sleepTime * time.Second)
		d.remainingTime -= sleepTime
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
