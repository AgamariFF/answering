package main

import (
	"answering/logger"
	"answering/tg"
	"fmt"
	"os"
)

func main() {
	os.Remove("QR.png")

	log, err := logger.NewLogger()
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		return
	}
	defer log.Close()

	_, cancelTg := tg.SetupTg(log)
	defer cancelTg()

	// ai.Handler(log)
}
