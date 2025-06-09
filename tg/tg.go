package tg

import (
	"answering/logger"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func Handler(log *logger.Logger) {
	profilePathTg := "./chrome_profile_tg"

	optionsTg := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("user-data-dir", profilePathTg),
		// chromedp.ProxyServer("45.8.211.64:80"),
		chromedp.Flag("headless", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		chromedp.Flag("enable-automation", false),
		// chromedp.Flag("disable-web-security", true),
		// chromedp.Flag("allow-running-insecure-content", true),
	)

	ctx, cancel := chromedp.NewContext(func() context.Context {
		ctx, _ := chromedp.NewExecAllocator(context.Background(), optionsTg...)
		return ctx
	}(),
		chromedp.WithLogf(func(string, ...interface{}) {}),
		chromedp.WithDebugf(func(string, ...interface{}) {}),
		chromedp.WithErrorf(func(string, ...interface{}) {}),
	)
	defer cancel()

	urlTg := "https://web.telegram.org/a"
	var screenBuffer []byte
	var exists bool

	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1500, 800),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.InfoLog.Println("Chrome to TG started")
			return nil
		}),
		chromedp.Navigate(urlTg),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
            document.body.innerText.includes("Log in to Telegram by QR Code")
        `, &exists),
	)
	if err != nil {
		log.ErrorLog.Println(err)
	}

	if !exists {
		log.InfoLog.Println("the entry has already been completed in Tg")
	}
	log.InfoLog.Println("login is required in Tg")

	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(1500, 800),
		chromedp.WaitVisible(`//*[@id="auth-qr-form"]/div/button[1]`),
		chromedp.Sleep(2*time.Second),
		chromedp.Screenshot("#auth-qr-form > div > div", &screenBuffer, chromedp.NodeVisible),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Scan QR in QR.png")
			os.WriteFile("QR.png", screenBuffer, 0644)
			log.InfoLog.Println("Made the ScreenShot with QR")
			return nil
		}),
		chromedp.WaitVisible(`//*[@id="LeftMainHeader"]`),
	)
	if err != nil {
		log.ErrorLog.Panicln("Error while performing the automation logic:", err)
	}
	os.Remove("QR.png")
	time.Sleep(6 * time.Second)

	var classAttr string

	err = chromedp.Run(ctx,
		chromedp.AttributeValue(`//*[@id="folders-container"]/div[1]/div[2]/ul/a[1]`, "class", &classAttr, &exists),
	)
	fmt.Println(classAttr)
}
