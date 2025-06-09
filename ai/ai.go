package ai

import (
	"answering/logger"
	"answering/models"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

func Handler(log *logger.Logger, incoming chan models.Message, outcoming chan models.Message, wg *sync.WaitGroup) {
	profilePath := "./chrome_profile_ai"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("user-data-dir", profilePath),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)

	allocCtx, cancel0 := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel1 := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(string, ...interface{}) {}),
		chromedp.WithDebugf(func(string, ...interface{}) {}),
		chromedp.WithErrorf(func(string, ...interface{}) {}),
	)

	defer cancel0()
	defer cancel1()

	url := "https://character.ai/chat/8XYC6I1tuVesOeifyRXkz6k0Q9tJxoLVXewJR1q5In4"
	var text string

	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1570, 730),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.InfoLog.Println("Chrome to Ai started")
			return nil
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Text(`/html`, &text),
	)
	if err != nil {
		log.ErrorLog.Println(err)
	}

	if strings.Contains(text, "Продолжить с Google") {
		fmt.Println("Необходима авторизация в AI для профиля: ", profilePath)
		log.InfoLog.Println("Необходима авторизация в AI для профиля: ", profilePath)
		time.Sleep(time.Minute)
		cancel0()
		cancel1()

	} else {
		fmt.Println("Авторизация в Ai не требуйется")
		log.InfoLog.Println("Авторизация в Ai не требуйется")
	}

	time.Sleep(5 * time.Second)
	wg.Done()
	wg.Wait()

	for {
		incomingMsg := <-incoming
		log.InfoLog.Println("AiHeabdler считал входящее сообщение: ", incomingMsg)
		var outcomingMsg models.Message
		outcomingMsg.ID = incomingMsg.ID
		err = chromedp.Run(ctx,
			chromedp.SendKeys(`//*[@id="chat-body"]/div[2]/div/div/div/div[1]/textarea`, incomingMsg.Text, chromedp.NodeVisible),
			chromedp.Click(`//*[@id="chat-body"]/div[2]/div/div/div/div[2]/button`, chromedp.NodeVisible),
			chromedp.Sleep(15*time.Second),
			chromedp.Text(`//*[@id="chat-messages"]/div[1]/div[1]/div/div/div[1]/div/div[1]/div[1]/div[2]/div[2]/div/div[1]`, &outcomingMsg.Text, chromedp.NodeVisible),
		)
		if err != nil {
			log.ErrorLog.Println(err)
			continue
		}
		if strings.HasSuffix(outcomingMsg.Text, ".") {
			outcomingMsg.Text = outcomingMsg.Text[:len(outcomingMsg.Text)-1]
		}
		log.InfoLog.Println("AiHeabdler обработал сообщение и пытается отправить ответ в oucoming: ", outcomingMsg)
		outcoming <- outcomingMsg
		log.InfoLog.Println("AiHeabdler отправил ответ в outcoming: ", outcomingMsg)
	}
}
