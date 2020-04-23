package push

import (
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Telegram implements the Telegram messenger
type Telegram struct {
	sync.Mutex
	bot   *tgbotapi.BotAPI
	chats map[int64]struct{}
	reply tgbotapi.Message
}

type telegramConfig struct {
	Token string
	Chats []int64
}

func init() {
	if err := tgbotapi.SetLogger(log.ERROR); err != nil {
		log.ERROR.Printf("telegram: %v", err)
	}
}

var modes = map[string]string{
	"Off": "off",
	"Now": "now",
	"Min": "min",
	"PV":  "pv",
}

// NewTelegramMessenger creates new pushover messenger
func NewTelegramMessenger(token string, chats []int64) *Telegram {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.FATAL.Fatal("telegram: invalid bot token")
	}

	m := &Telegram{
		bot:   bot,
		chats: make(map[int64]struct{}),
	}

	for _, chat := range chats {
		m.chats[chat] = struct{}{}
	}

	go m.trackChats()

	return m
}

// trackChats captures ids of all chats that bot participates in
func (m *Telegram) trackChats() {
	conf := tgbotapi.NewUpdate(0)
	conf.Timeout = 1000

	updates, err := m.bot.GetUpdatesChan(conf)
	if err != nil {
		log.ERROR.Printf("telegram: %v", err)
	}

	for msg := range updates {
		// process callback
		if msg.CallbackQuery != nil {
			m.processCallback(msg.CallbackQuery)
			continue
		}
		// abort if not from chat
		if msg.Message == nil || msg.Message.Chat == nil {
			continue
		}

		m.Lock()
		if _, ok := m.chats[msg.Message.Chat.ID]; ok {
			m.processMessage(msg.Message)
		} else {
			log.INFO.Printf("telegram: new chat id: %d", msg.Message.Chat.ID)
		}
		m.Unlock()
	}
}

func (m *Telegram) processCallback(msg *tgbotapi.CallbackQuery) {
	log.INFO.Printf("%+v", msg)

	cb := tgbotapi.NewCallback(msg.ID, "done")
	if _, err := m.bot.AnswerCallbackQuery(cb); err != nil {
		log.ERROR.Print(err)
	}

	var mode string
	for k, v := range modes {
		if v == msg.Data {
			mode = k
			break
		}
	}

	if mode == "" {
		log.ERROR.Printf("telegram: invalid callback request %v", msg)
		return
	}

	text := fmt.Sprintf("Changed mode to %s", mode)
	edit := tgbotapi.NewEditMessageText(m.reply.Chat.ID, m.reply.MessageID, text)
	if _, err := m.bot.Send(edit); err != nil {
		log.ERROR.Print(err)
	}
}

// processUpdate replies to updates received
func (m *Telegram) processMessage(msg *tgbotapi.Message) {
	log.INFO.Printf("%+v", msg)

	if strings.HasPrefix(strings.ToLower(msg.Text), "/") {
		buttons := make([]tgbotapi.InlineKeyboardButton, len(modes))
		for k, v := range modes {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(k, v))
		}

		log.INFO.Printf("%+v", buttons)

		reply := tgbotapi.NewMessage(msg.Chat.ID, "Select mode:")
		ikm := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))

		// ikm = tgbotapi.NewInlineKeyboardMarkup(
		// 	tgbotapi.NewInlineKeyboardRow(
		// 		tgbotapi.NewInlineKeyboardButtonData("Off", "off"),
		// 		tgbotapi.NewInlineKeyboardButtonData("Now", "now"),
		// 		tgbotapi.NewInlineKeyboardButtonData("Min", "min"),
		// 		tgbotapi.NewInlineKeyboardButtonData("PV", "pv"),
		// 	),
		// )

		reply.ReplyMarkup = ikm

		if reply, err := m.bot.Send(reply); err != nil {
			log.ERROR.Print(err)
		} else {
			m.reply = reply
		}
	}
}

// Send sends to all receivers
func (m *Telegram) Send(event Event, title, msg string) {
	m.Lock()
	for chat := range m.chats {
		log.TRACE.Printf("telegram: sending to %d", chat)

		msg := tgbotapi.NewMessage(chat, msg)
		if _, err := m.bot.Send(msg); err != nil {
			log.ERROR.Print(err)
		}
	}
	m.Unlock()
}
