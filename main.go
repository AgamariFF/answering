package main

import (
	"answering/ai"
	"answering/logger"
	"answering/models"
	"answering/tg"
	"fmt"
	"os"
	"sync"
	"time"
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
	deleteId := make(chan string)

	go tg.HandlerCheckMsg(log, newMessage, ctxTg)

	defer cancelTg()

	fmt.Println("Программа запущена")

	go func() {
		for true {
			select {
			case NewId := <-newMessage:
				if _, exists := IdChats[NewId]; exists {
					log.InfoLog.Println("Чат уже обрабатывается")
				} else {
					go dialog(log, NewId, "https://character.ai/chat/8XYC6I1tuVesOeifyRXkz6k0Q9tJxoLVXewJR1q5In4", deleteId)
					IdChats[NewId] = struct{}{}
					log.InfoLog.Println("Чат запущен в обработку")
				}
				time.Sleep(5 * time.Second)
			case IdChatDelete := <-deleteId:
				delete(IdChats, IdChatDelete)
			}
		}
	}()

	var end string
	fmt.Scan(&end)
}

func dialog(log *logger.Logger, chat_id string, ai_url string, delete_chat_id chan string) {
	incoming := make(chan models.Message)
	outcoming := make(chan models.Message)
	monitoringChannel := make(chan bool)

	var waitSetupDialog sync.WaitGroup
	waitSetupDialog.Add(2)

	tg_url := "https://web.telegram.org/a/#" + chat_id

	ctxTg, cancelTg := tg.SetupTg(log, tg_url)
	ctxAi, cancelAi0, cancelAi1 := ai.SetupAi(log, &waitSetupDialog)

	defer cancelTg()
	defer cancelAi0()
	defer cancelAi1()

	go tg.HandlerDialog(log, incoming, outcoming, monitoringChannel, ctxTg)
	go ai.Handler(log, incoming, outcoming, ctxAi)

	waitSetupDialog.Wait()
	fmt.Println("Чат по id = ", chat_id, "запущен")

	// Таймер для отслеживания времени бездействия
	inactivityTimeout := 5 * time.Minute
	timer := time.NewTimer(inactivityTimeout)

	for {
		select {
		case <-timer.C:
			log.InfoLog.Println("Таймаут бездействия. Удаляем чат с id:" + chat_id)
			delete_chat_id <- chat_id
			return

		case _ = <-monitoringChannel:
			// Если пришло новое сообщение, сбрасываем таймер
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(inactivityTimeout)

		case <-ctxTg.Done():
			// Если контекст завершен, прекращаем работу
			delete_chat_id <- chat_id
			log.InfoLog.Println("Контекст Telegram завершен. Завершаем работу.")
			return

		case <-ctxAi.Done():
			// Если контекст AI завершен, прекращаем работу
			delete_chat_id <- chat_id
			log.InfoLog.Println("Контекст AI завершен. Завершаем работу.")
			return
		}
	}

}
