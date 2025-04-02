package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	tgbotapi "fenrir/tgbotapi"

	"github.com/recoilme/pudge"
)

// ******** Settings **********
var operatorChatId int64 = -0

// var settingsThread string = "498"
var TOKEN string = "???????????????????????"

// *****************************

// память
var q = make(map[string]string)

// https://core.telegram.org/bots/api#
// структуры топиков
type ForumTopicStruct struct {
	MessageThreadId   int64  `json:"message_thread_id"`           //Unique identifier of the forum topic
	Name              string `json:"name"`                        // of the topic
	IconColor         int64  `json:"icon_color"`                  //Color of the topic icon in RGB format
	IconCustomEmojiId string `json:"string icon_custom_emoji_id"` // 	Optional. Unique identifier of the custom emoji shown as the topic icon
}

func create_inline_keyboard(q map[string]string) [][]tgbotapi.InlineKeyboardButton {
	inline_keyboard := make([][]tgbotapi.InlineKeyboardButton, int8(len(q)/4)+1)
	n, k := 0, 0
	for key, value := range q {
		inline_keyboard[n] = append(inline_keyboard[n], tgbotapi.InlineKeyboardButton{
			Text:         key,
			CallbackData: &value,
		})
		k += 1
		if k%3 == 0 {
			n += 1
			k = 0
		}
	}
	return inline_keyboard
}

func sendToThread(bot *tgbotapi.BotAPI, chatId int64, message_thread_id int64, text string, inline_keyboard [][]tgbotapi.InlineKeyboardButton) error {
	var messageThread = make(tgbotapi.Params)
	messageThread["chat_id"] = fmt.Sprint(chatId, "_", message_thread_id)
	messageThread["message_thread_id"] = fmt.Sprint(message_thread_id)
	messageThread["text"] = text
	messageThread["parse_mode"] = "HTML"

	if inline_keyboard != nil {
		ikm := make(map[string]any)
		ikm["inline_keyboard"] = inline_keyboard
		jsonBytes, err := json.Marshal(ikm)
		if err != nil {
			return err
		}
		messageThread["reply_markup"] = string(jsonBytes)
	}

	_, err := bot.MakeRequest("sendMessage", messageThread)
	if err != nil {
		return err
	}
	return nil
}

func forwardToThread(bot *tgbotapi.BotAPI, chatId int64, from_chat_id int64, message_thread_id int64, message_id int) error {
	var messageThread = make(tgbotapi.Params)
	messageThread["chat_id"] = fmt.Sprint(chatId, "_", message_thread_id)
	messageThread["message_thread_id"] = fmt.Sprint(message_thread_id)
	messageThread["message_id"] = fmt.Sprint(message_id)
	messageThread["from_chat_id"] = fmt.Sprint(from_chat_id)

	_, err := bot.MakeRequest("forwardMessage", messageThread)
	if err != nil {
		return err
	}
	return nil
}

func telegramBot() {

	defer pudge.CloseAll()

	//Создаем бота
	bot, err := tgbotapi.NewBotAPI(TOKEN)
	if err != nil {
		panic(err)
	}

	//bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	q["*"] = "शून्यता"

	//Получаем обновления от бота
	updates := bot.GetUpdatesChan(u)
	for update := range updates {

		if update.CallbackQuery != nil {
			// https://t.me/c/2130011723/498/499
			var clientChatId int64
			pudge.Get("topicdb", update.CallbackQuery.Message.MessageThreadId, &clientChatId)

			if strings.HasPrefix(update.CallbackQuery.Data, "https://t.me/c/") {
				var messageId int
				d := strings.Split(strings.TrimPrefix(update.CallbackQuery.Data, "https://t.me/c/"), "/")
				messageId, err := strconv.Atoi(d[2])
				if err == nil {
					//fmt.Printf("messageId=%d, type: %T\n", messageId, messageId)
					copymsg := tgbotapi.NewCopyMessage(clientChatId, update.CallbackQuery.Message.Chat.ID, messageId)
					bot.Send(copymsg)

				}

				err2 := forwardToThread(bot, operatorChatId, operatorChatId, update.CallbackQuery.Message.MessageThreadId, messageId)
				if err2 != nil {
					fmt.Println(err2)
				}

			} else {
				msg := tgbotapi.NewMessage(clientChatId, update.CallbackData())

				bot.Send(msg)

				err := sendToThread(bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageThreadId, update.CallbackData(), nil)
				if err != nil {
					fmt.Println(err)
				}
			}

			// удаление сообщения после отправки
			msg2 := tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)
			bot.Send(msg2)

		}

		if update.Message == nil {
			continue
		}

		//пользователи!
		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" && update.Message.Chat.ID != operatorChatId {
			switch update.Message.Text {
			case "/start":

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать! Создаю новый топик. Можете задать вопрос или обратиться с просьбой.")
				bot.Send(msg)
				// создание нового топика
				var topic = make(tgbotapi.Params)
				topic["chat_id"] = fmt.Sprint(operatorChatId)
				topic["name"] = fmt.Sprint(update.Message.Chat.UserName) //// name
				resp, err := bot.MakeRequest("createForumTopic", topic)
				if err != nil {
					fmt.Println(err)
				}
				// отправка первого сообщения в новый топик
				var forumtopic ForumTopicStruct
				err = json.Unmarshal(resp.Result, &forumtopic)
				if err != nil {
					fmt.Println(err)
				}
				msg_to_thread := "Новый пользователь зарегистрирован! 🚀 @" + update.Message.Chat.UserName + ":  " + update.Message.Chat.FirstName + " " + update.Message.Chat.LastName
				err = sendToThread(bot, operatorChatId, forumtopic.MessageThreadId, msg_to_thread, nil)
				if err != nil {
					fmt.Println(err)
				}
				// запись в базу
				pudge.Set("topicdb", update.Message.Chat.ID, forumtopic.MessageThreadId)
				pudge.Set("topicdb", forumtopic.MessageThreadId, update.Message.Chat.ID)

			default:
				var MessageThreadId int64
				pudge.Get("topicdb", update.Message.Chat.ID, &MessageThreadId)
				// перенаправляем сообщение в топик
				txt := update.Message.Text + "     << @" + update.Message.From.UserName
				err := sendToThread(bot, operatorChatId, MessageThreadId, txt, nil)
				if err != nil {
					fmt.Println(err)
				}
			}

		} else

		//операторы!
		{
			var clientChatId int64
			var msg tgbotapi.MessageConfig
			err := pudge.Get("topicdb", update.Message.MessageThreadId, &clientChatId)
			if err != nil {
				fmt.Println(err)
			}

			//	Управление коммандной памятью:
			//		"/k" - показать всё в виде инлайн-клавиатуры,
			//		"/rm" - удалить всЁ,
			//		"*(название комманды)" - отправить
			//  	"/add(пробел)*(название)$(содержание или ссылка на сообщение)" - добавить новую комманду
			if strings.HasPrefix(update.Message.Text, "/add") {
				sl := strings.Split(strings.TrimPrefix(update.Message.Text, "/add "), "$")
				if strings.HasPrefix(sl[0], "*") {
					q[sl[0]] = sl[1]
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Добавлена новая команда: "+sl[0]+"  ->  "+sl[1])
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "🛇 > /add *...$...")

				}
			} else if strings.HasPrefix(update.Message.Text, "/k") {
				if len(q) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "🛇 No data")
				} else {
					delmsg := tgbotapi.NewDeleteMessage(operatorChatId, update.Message.MessageID)
					bot.Send(delmsg)

					inline_keyboard := create_inline_keyboard(q)
					err := sendToThread(bot, operatorChatId, update.Message.MessageThreadId, ".", inline_keyboard)
					if err != nil {
						fmt.Println(err)
					}

				}
			} else if strings.HasPrefix(update.Message.Text, "/rm") {
				for k := range q {
					delete(q, k)
				}
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "🚮 Deleted")
			} else if strings.HasPrefix(update.Message.Text, "*") {
				msg = tgbotapi.NewMessage(clientChatId, q[update.Message.Text])
			} else {

				member_info, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: operatorChatId, UserID: update.Message.From.ID}})
				if err != nil {
					fmt.Println(err)
				}

				// подпись
				var sign string
				if member_info.CustomTitle == "" {
					sign = fmt.Sprint("<i><b>Пишет ", update.Message.From.FirstName, "_", update.Message.From.LastName, ": </b></i>", update.Message.Text)
				} else {
					sign = fmt.Sprint("<i><b>Пишет ", member_info.CustomTitle, ": </b></i>", update.Message.Text)
				}

				msg = tgbotapi.NewMessage(clientChatId, sign)
				msg.ParseMode = "HTML"
			}
			// - #end -

			bot.Send(msg)

		}
	}
}

func main() {
	time.Sleep(3 * time.Second)
	fmt.Println("start bot " + time.Now().String())
	//Вызываем бота
	telegramBot()
}
