package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func GetTemp(city, w_api string) string {

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
		return fmt.Sprint("Не найден город с таким названием!")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var w Weather
	if err := json.Unmarshal(body, &w); err != nil {
		slog.Error(err.Error())
		return fmt.Sprint("Произошла неизвестная ошибка. . .")
	}

	if w.Main.Temp == 0. && w.Main.FeelsLike == 0. && w.Wind.Speed == 0. {
		slog.Warn("Нет такого города!")
		return fmt.Sprint("Не найден город с таким названием!")
	}

	return fmt.Sprintf("Прогноз погоды в %s следующий:\n\tТемпература: %f\n\tОщущается как: %f\n\tСкорость ветра: %f", w.Name, w.Main.Temp, w.Main.FeelsLike, w.Wind.Speed)

}

func main() {

	bot_api := os.Getenv("BOT_API")
	w_api := os.Getenv("WEATHER_API")
	bot, err := tgbotapi.NewBotAPI(bot_api)
	if err != nil {
		slog.Error(err.Error())
		fmt.Println(err)
		return
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		slog.Info(fmt.Sprintf("Get new message: %s\nfrom %s", update.Message.Text, update.Message.From))

		if update.Message.Command() == "start" {
			text := fmt.Sprintf("Привет, %s! Я бот, который выдает прогноз погоды городов. Ты можешь написать название города, а я выдам тебе актуальную температуру в данном городе!", update.Message.From.FirstName)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			bot.Send(msg)
			continue
		}

		text := GetTemp(update.Message.Text, w_api)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		bot.Send(msg)

	}

}
