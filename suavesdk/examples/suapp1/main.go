package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/suavesdk"
)

func main() {
	s := &Suapp{}

	suapp := suavesdk.NewSuapp(
		suavesdk.WithFunction(s),
	)

	topic := suapp.NewTopic("test")
	topic.Subscribe(func(data []byte) {
		s.handleData(data)
	})
}

type Suapp struct {
	topic *suavesdk.Topic
}

func (s *Suapp) Do() (uint64, error) {
	// public topic to redis
	s.topic.Publish([]byte("hello world"))

	fmt.Println("__ FUNCTION CALLED!!! __")
	return 1, nil
}

func (s *Suapp) handleData(data []byte) {
	// store data on redis async
}
