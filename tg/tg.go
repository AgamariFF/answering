package tg

import (
	"answering/internal"
	"answering/logger"
	"answering/models"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func FindFreeProfile(log *logger.Logger) string {
	var profilePath string
	for i := 0; i < 100; i++ {
		profilePath = "./chrome_profile_tg_" + strconv.Itoa(i) + "/"
		if !internal.IsBrowserRunning(profilePath) {
			break
		}
		if i == 99 {
			panic("Не удалось найти свободный профиль")
		}
	}
	if err := os.MkdirAll(profilePath, 0755); err != nil {
		panic(err)
	}
	log.InfoLog.Println("Свободный путь для Tg: ", profilePath)

	return profilePath
}

func SetupTg(log *logger.Logger, urlTg string) (context.Context, context.CancelFunc) {
	profilePathTg := FindFreeProfile(log)

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

	if exists {
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
	} else {
		log.InfoLog.Println("the entry has already been completed in Tg")
	}

	time.Sleep(3 * time.Second)

	return ctx, cancel
}

func HandlerCheckMsg(log *logger.Logger, NewChatId chan string, ctx context.Context) {
	var id, author string
	var exists bool
	for i := 0; true; i++ {
		time.Sleep(5 * time.Second)
		count := 2 + i%5
		xpath := `//*[@id="LeftColumn-main"]/div[2]/div/div[2]/div/div[2]/div[` + strconv.Itoa(count) + `]/a`
		xpathText := xpath + `/div[3]/div[2]/div/div`
		xpathAuthor := xpath + `/div[3]/div[1]/div[1]/h3`

		err := chromedp.Run(ctx,
			chromedp.EvaluateAsDevTools(fmt.Sprintf(`
			document.evaluate('%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue !== null
		`, xpathText), &exists),
		)
		if err != nil {
			log.ErrorLog.Println("Failed to find unread: " + err.Error())
		}
		if !exists {
			continue
		}
		log.InfoLog.Println("Найдено непрочитанное сообщение")

		var ids []string
		err = chromedp.Run(ctx,
			chromedp.AttributeValue(xpath, "href", &id, nil, chromedp.BySearch),
			chromedp.Text(xpathAuthor, &author),
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('[id^="message-"]'))
				.map(el => el.id)
				.filter(id => /^message-\d+$/.test(id))
				`, &ids),
			chromedp.Navigate("https://web.telegram.org/a"),
		)
		if err != nil {
			log.ErrorLog.Println(err)
		}
		NewChatId <- id[1:len(id)]
	}
}

func HandlerDialog(log *logger.Logger, incoming chan models.Message, outcoming chan models.Message, monitoringChannel chan bool, ctx context.Context) {
	var outcomingMsg models.Message
	var n int
	maxID := 0
	lastId := 0
	lastMsg := models.Message{Text: "", ID: ""}

	time.Sleep(time.Second * 5)

	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1500, 800),
		chromedp.Evaluate(
			`
    const container = document.querySelector('.MessageList'); // Замените на реальный селектор
    if (container) {
        container.scrollTop = container.scrollHeight;
    } else {
        console.error("Контейнер для прокрутки не найден");
    }
`, nil),
	)
	if err != nil {
		log.ErrorLog.Println("Ошибка изменения разрешения или прокручивания")
	}

	maxID = findMaxMessageId(log, ctx)

	for n = maxID; n >= lastId; n-- {
		var outMsg bool
		var exists bool
		var classAttr string
		xpath := fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/div`, n)
		// xpath := fmt.Sprintf(`//*[@data-mid="%d"]/div[3]/div/div[1]/div`, n)

		// Проверка существования элемента
		err := chromedp.Run(ctx,
			chromedp.Evaluate(
				fmt.Sprintf(`
                !!document.evaluate('%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue
            `, xpath),
				&exists,
			),
		)
		if err != nil || !exists {
			continue
		} else {
			err = chromedp.Run(ctx,
				chromedp.AttributeValue(xpath, "class", &classAttr, &exists),
			)
			if err != nil {
				log.ErrorLog.Println(err)
				continue
			}
		}

		log.InfoLog.Println("Проверяю сообщение с id=", n, "\tАттрибуты: ", classAttr)

		substrings := []string{"with-outgoing-icon", "own"}
		for _, substing := range substrings {
			if strings.Contains(classAttr, substing) {
				outMsg = true
			}
		}
		if outMsg {
			log.InfoLog.Println("Это исходящее сообщение")
			outMsg = false
			continue
		}
		lastMsg.ID = strconv.Itoa(n)
		log.InfoLog.Println("Последнее входящее сообщение в этом чате имеет id = ", n)
		lastId = n
		break
	}
	fmt.Println("Начал считывать сообщения в Tg")

	var skipMsg bool
	lastId--

	for {
		select {
		case outcomingMsg = <-outcoming:
			log.InfoLog.Println("Считано сообщение из outcoming: ", outcomingMsg)
			err := chromedp.Run(ctx,
				chromedp.SendKeys(`//*[@id="editable-message-text"]`, outcomingMsg.Text, chromedp.NodeVisible),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.Click(`//*[@id="MiddleColumn"]/div[4]/div[3]/div/div[2]/div[1]/button`, chromedp.NodeVisible),
			)
			if err != nil {
				log.ErrorLog.Println(err)
			}
			log.InfoLog.Println("Outcoming message: ", outcomingMsg.Text, "has been sended")
		default:
			maxID = 0

			maxID = findMaxMessageId(log, ctx)

			// Шаг 2: Собрать текст из элементов без with-outgoing-icon
			lastId, err := strconv.Atoi(lastMsg.ID)
			if err != nil {
				log.ErrorLog.Println("Ошибка при конвертации ID сообщения в число: " + err.Error())
			}

			var message string
			for n := maxID; n <= lastId; n-- {
				var outMsg bool
				var exists bool
				var classAttr string
				xpath := fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/div`, n)
				// xpath := fmt.Sprintf(`//*[@data-mid="%d"]/div[3]/div/div[1]/div`, n)

				err = chromedp.Run(ctx,
					chromedp.Evaluate(
						fmt.Sprintf(`
                !!document.evaluate('%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue
            `, xpath),
						&exists,
					),
				)

				if err != nil || !exists {
					continue
				}

				err = chromedp.Run(ctx,
					chromedp.AttributeValue(xpath, "class", &classAttr, &exists),
				)

				if err != nil {
					log.ErrorLog.Println(err)
					continue
				}

				substrings := []string{"with-outgoing-icon", "own"}
				for _, substing := range substrings {
					if strings.Contains(classAttr, substing) {
						outMsg = true
					}
				}

				if outMsg {
					outMsg = false
					continue
				}

				if lastId == n {
					skipMsg = true
					break
				}
				lastMsg.ID = strconv.Itoa(n)

				log.InfoLog.Println("Обнаружено новое входящее сообщение")

				log.InfoLog.Println("Сообщение входящее, его аттрибуты: " + classAttr)

				if strings.Contains(classAttr, "message-subheader") { // это ответ на сообщение
					xpath2 := fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/div[2]`, n)
					err = chromedp.Run(ctx,
						chromedp.Text(xpath2, &message, chromedp.BySearch),
					)
					if err != nil {
						log.ErrorLog.Fatalln(err)
					}
					if strings.Contains(classAttr, "Audio") { // Пришло гс ответом на сообщение
						message = convertVoice(ctx, n, log, message, 1)
					}
					if strings.Contains(classAttr, "RoundVideo") { // Пришел кружок ответом на сообщение
						message = convertVoice(ctx, n, log, message, 2)
					} else { // Текстовое сообщение ответом на сообщение
						message = message[:len(message)-6]
					}
					break
				} else { // это не ответ на сообщение
					err = chromedp.Run(ctx,
						chromedp.Text(xpath, &message, chromedp.BySearch),
					)
					if strings.Contains(classAttr, "Audio") { // Пришло гс
						message = convertVoice(ctx, n, log, message, 1)
					}
					if strings.Contains(classAttr, "RoundVideo") { // Пришел кружок
						message = convertVoice(ctx, n, log, message, 2)
					} else { // Текстовое сообщение
						message = message[:len(message)-6]
					}
					if err != nil {
						log.ErrorLog.Fatalln(err)
					}
					break
				}
			}
			if skipMsg {
				skipMsg = false
				continue
			}
			fmt.Println("Обнаружено новое сообщение: ", message)
			lastMsg.Text = message
			incoming <- lastMsg
			monitoringChannel <- true

			log.InfoLog.Println("В incoming отправлено: ", lastMsg)
			time.Sleep(time.Second)
		}
		time.Sleep(time.Second)
	}
}

func findMaxMessageId(log *logger.Logger, ctx context.Context) int {
	const jsScript = `
(() => {
    const elements = document.querySelectorAll('[data-message-id]');
    let maxID = 0;
    elements.forEach(el => {
        const id = parseInt(el.getAttribute('data-message-id'), 10);
        if (id > maxID) maxID = id;
    });
    return maxID;
})();
`

	maxID := 0
	err := chromedp.Run(ctx,
		chromedp.Evaluate(jsScript, &maxID))
	log.InfoLog.Println("Максимальное Id сообщения: ", maxID)

	if err != nil {
		log.ErrorLog.Fatalln("Ошибка поиска максимального n:", err)
	}

	if maxID == 0 {
		var pageContent string
		err := chromedp.Run(ctx,
			chromedp.OuterHTML("html", &pageContent),
		)
		if err != nil {
			log.ErrorLog.Println("Элементы не найдены или некорректные ID")
			log.ErrorLog.Panic("Ошибка при получении HTML-кода страницы:", err)
		}
		log.InfoLog.Println("HTML-код страницы:", pageContent)
		log.ErrorLog.Panic("Элементы не найдены или некорректные ID; Код старницы:\n")
	}

	return maxID
}

func convertVoice(ctx context.Context, id int, log *logger.Logger, msgTime string, typeMsg int) string {
	sleepTime, err := parseDuration(msgTime)
	var xpath string
	if err != nil {
		log.ErrorLog.Println(err)
		sleepTime = 30 * time.Second
	}
	switch typeMsg {
	case 1:
		xpath = fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/div/div[2]/div/button`, id)
	case 2:
		xpath = fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/div//button`, id)
	}
	xpath2 := fmt.Sprintf(`//*[@id="message-%d"]/div[3]/div/div[1]/p`, id)
	var message string
	err = chromedp.Run(ctx,
		chromedp.Click(xpath, chromedp.NodeVisible),
		chromedp.Sleep(sleepTime),
		chromedp.Text(xpath2, &message, chromedp.NodeVisible, chromedp.BySearch),
	)
	if err != nil {
		log.ErrorLog.Println(err)
		return ""
	}
	return message
}

func parseDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, errors.New("некорректный формат, требуется 'мин:сек', получен: " + s)
	}

	min, err := strconv.Atoi(parts[0])
	if err != nil || min < 0 {
		return 0, errors.New("некорректные минуты")
	}

	sec, err := strconv.Atoi(parts[1])
	if err != nil || sec < 0 || sec >= 60 {
		return 0, errors.New("некорректные секунды")
	}

	return time.Duration(min)*time.Minute + time.Duration(sec)*time.Second, nil
}

func processMessage(input, author string) string {
	// Убираем время в формате HH:MM
	re := regexp.MustCompile(`\d{2}:\d{2}`)
	input = re.ReplaceAllString(input, "")

	// Заменяем все переносы строк на точки
	input = strings.ReplaceAll(input, "\n", ". ")

	// Убираем лишние точки подряд и пробелы вокруг
	input = regexp.MustCompile(`\.{2,}`).ReplaceAllString(input, ".")
	input = strings.Trim(input, ". ")

	// Добавляем префикс "Отправитель: "
	return "Отправитель: " + author + ". Сообщение: " + input
}
