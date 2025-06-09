package main

import (
	"answering/ai"
	"answering/logger"
	"answering/models"
	"answering/tg"
	"fmt"
	"os"
	"sync"
)

func main() {
	os.Remove("QR.png")

	log, err := logger.NewLogger()
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		return
	}
	defer log.Close()

	incoming := make(chan models.Message)
	outcoming := make(chan models.Message)

	var waitSetup sync.WaitGroup
	waitSetup.Add(2)

	go tg.Handler(log, incoming, outcoming, &waitSetup)

	go ai.Handler(log, incoming, outcoming, &waitSetup)

	var end string
	fmt.Scan(&end)
}
