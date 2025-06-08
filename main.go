package main

import (
	"answering/logger"
	"fmt"
)

func main() {
	log, err := logger.NewLogger()
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		return
	}
	defer log.Close()

}
