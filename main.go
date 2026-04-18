package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/go-redis/v9"
)

type Weather struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	Name string `json:"name"`
}

func GetTemp(city, w_api string) (string, *Weather) {

	base_url := "https://api.openweathermap.org/data/2.5/weather?"

	params := url.Values{}
	params.Add("q", city)
	params.Add("appid", w_api)
	params.Add("units", "metric")
	params.Add("lang", "ru")

	url := base_url + params.Encode()
	resp, err := http.Get(url)
	if err != nil {
		slog.Warn(err.Error())
		return fmt.Sprint("Не найден город с таким названием!"), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var w Weather
	if err := json.Unmarshal(body, &w); err != nil {
		slog.Error(err.Error())
		return fmt.Sprint("Произошла неизвестная ошибка. . ."), nil
	}

	if w.Main.Temp == 0. && w.Main.FeelsLike == 0. && w.Wind.Speed == 0. {
		slog.Warn("Нет такого города!")
		return fmt.Sprint("Не найден город с таким названием!"), nil
	}

	return fmt.Sprintf("Прогноз погоды в %s следующий:\n\tТемпература: %f\n\tОщущается как: %f\n\tСкорость ветра: %f", w.Name, w.Main.Temp, w.Main.FeelsLike, w.Wind.Speed), &w

}

func main() {

	ctx := context.Background()

	RedisAddr := os.Getenv("REDIS_ADDR")
	rdb := redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: "",
		DB:       0,
	})

	bot_api := os.Getenv("BOT_API")
	w_api := os.Getenv("WEATHER_API")
	bot, err := tgbotapi.NewBotAPI(bot_api)
	if err != nil {
		slog.Error(err.Error())
		fmt.Println(err)
		return
	}

	wishes := []string{
		"https://i.pinimg.com/736x/5c/1c/67/5c1c6782c8d795a0a10886faf9a41c7b.jpg",
		"https://i.pinimg.com/736x/ab/98/fd/ab98fdc657a92362215600f7a025a74a.jpg",
		"https://i.pinimg.com/736x/02/7b/54/027b54cd912005362204bbf3a233c242.jpg",
		"https://i.pinimg.com/736x/de/34/9b/de349b4408e3e86301db25ee7032620e.jpg",
		"https://i.pinimg.com/1200x/ed/91/2d/ed912d1d5d0017f9dc0b09d0bcc77880.jpg",
		"https://i.pinimg.com/736x/5e/a1/b5/5ea1b5eeed5ff41d4cbb587754e22cf8.jpg",
		"https://i.pinimg.com/1200x/1d/f2/f2/1df2f2fe61dffc0f9fc80f8e6efdb391.jpg",
		"https://i.pinimg.com/736x/6e/eb/a9/6eeba90e5c9ea314c7a1c46026c6fcd9.jpg",
		"https://i.pinimg.com/736x/13/82/9b/13829b20ccf753efcc380e1fcbb3e89f.jpg",
	}

	weather := map[string]string{
		"cold": "https://i.pinimg.com/736x/e2/ad/6a/e2ad6ab1cc68cc8ac2420c7768f92b09.jpg",
		"calm": "https://i.pinimg.com/736x/93/fc/82/93fc82f469336a839cd8b01608108442.jpg",
		"worm": "https://i.pinimg.com/736x/cd/f9/1b/cdf91b03cd5b0790de1e5e06b55c371a.jpg",
		"hot":  "https://i.pinimg.com/736x/e8/1f/cf/e81fcf55a94936ef6bd5cf3e81319438.jpg",
		"wind": "https://i.pinimg.com/736x/43/e1/9f/43e19f8fa32dadf0d38aa714dc3552a5.jpg",
		"idk":  "https://i.pinimg.com/736x/e8/51/55/e851556c794af699afae9095e7e07f58.jpg",
	}

	signal_ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)
	tasks := make(chan tgbotapi.Update, 100)
	const workers = 3
	wg := sync.WaitGroup{}

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Старт"},
		{Command: "help", Description: "Помощь"},
		{Command: "wish", Description: "Пожелание на день"},
	}
	config := tgbotapi.NewSetMyCommands(commands...)
	bot.Request(config)

	for i := 0; i < workers; i++ {

		wg.Add(1)
		go func() {

			defer wg.Done()
			for update := range tasks {

				if update.Message != nil {
					slog.Info(fmt.Sprintf("Get new message: %s\nfrom %s", update.Message.Text, update.Message.From))
					if update.Message.Command() == "start" || update.Message.Command() == "help" {
						text := fmt.Sprintf("Привет, %s! Я бот, который выдает прогноз погоды городов. Ты можешь написать название города, а я выдам тебе актуальную температуру в данном городе! Так же с недавних пор у бота появилась новая функция. Он может отправить тебе пожелание на день, для этого надо написать команду \"/wish\"", update.Message.From.FirstName)
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
						bot.Send(msg)
					} else if update.Message.Command() == "wish" {
						text := "Вот твое пожелание на день!"
						index := rand.Intn(len(wishes))
						photo := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FileURL(wishes[index]))
						photo.Caption = text
						bot.Send(photo)
					} else {
						var text string
						var w *Weather
						index := "calm"
						check := 1
						if exists, _ := rdb.Exists(ctx, strings.ToLower(update.Message.Text)).Result(); exists == 1 {
							check = 0
							data, err := rdb.Get(ctx, update.Message.Text).Bytes()
							if err == redis.Nil {
								slog.Warn(err.Error())
								text = fmt.Sprint("Не найден город с таким названием!")
							} else if err != nil {
								slog.Error(err.Error())
								check = 1
							} else {
								if err := json.Unmarshal(data, &w); err != nil {
									slog.Error(err.Error())
								} else {
									text = fmt.Sprintf("Прогноз погоды в %s следующий:\n\tТемпература: %f\n\tОщущается как: %f\n\tСкорость ветра: %f", w.Name, w.Main.Temp, w.Main.FeelsLike, w.Wind.Speed)
								}
							}
						}
						if check == 1 {
							text, w = GetTemp(update.Message.Text, w_api)
							if w != nil {
								data, err := json.Marshal(&w)
								if err != nil {
									slog.Error(err.Error())
								} else if err := rdb.Set(ctx, strings.ToLower(update.Message.Text), data, 300*time.Second).Err(); err != nil {
									slog.Error(err.Error())
								} else {
									slog.Info("Weather cached success %s", update.Message.Text)
								}
							}
						}
						if w == nil {
							index = "idk"
						} else if w.Wind.Speed > 10 {
							index = "wind"
						} else if w.Main.Temp <= 0 {
							index = "cold"
						} else if w.Main.Temp > 16 && w.Main.Temp <= 28 {
							index = "worm"
						} else if w.Main.Temp > 28 {
							index = "hot"
						}
						photo := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FileURL(weather[index]))
						photo.Caption = text
						bot.Send(photo)
					}

				}

			}

		}()

	}
	fmt.Println("Start check updates")
	go func() {
		for update := range updates {

			select {
			case <-signal_ctx.Done():
				break
			default:
				tasks <- update
			}

		}
	}()

	<-signal_ctx.Done()
	close(tasks)
	fmt.Println("wait until workers are working")
	wg.Wait()
	fmt.Println("GG")

}
