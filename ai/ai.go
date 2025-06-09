package ai

import (
	"answering/logger"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func Handler(log *logger.Logger) {
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

	url := "https://character.ai/"
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
}
