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

	IdChats := make(map[string]struct{})

	ctxTg, cancelTg := tg.SetupTg(log, "https://web.telegram.org/a")
	newMessage := make(chan string)
	go tg.HandlerCheckMsg(log, newMessage, ctxTg)

	defer cancelTg()

	for true {
		NewId := <-newMessage
	}

	var end string
	fmt.Scan(&end)
}

func dialog(log *logger.Logger, chat_id int, ai_url string) bool {
	incoming := make(chan models.Message)
	outcoming := make(chan models.Message)

	var waitSetupDialog sync.WaitGroup
	waitSetupDialog.Add(2)

	ctxTg, cancelTg := tg.SetupTg(log, &waitSetupDialog)
	ctxAi, cancelAi0, cancelAi1 := ai.SetupAi(log, &waitSetupDialog)

	defer cancelTg()
	defer cancelAi0()
	defer cancelAi1()

	go tg.Handler(log, incoming, outcoming, ctxTg)
	go ai.Handler(log, incoming, outcoming, ctxAi)

	waitSetupDialog.Wait()
	fmt.Println("Чат по id = ", chat_id, "запущен")
}
