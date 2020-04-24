package push

import (
	"fmt"
	"strings"
	"sync"

	"github.com/andig/evcc/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	cmdMode   = "mode"
	cmdStatus = "status"
)

// Telegram implements the Telegram messenger
type Telegram struct {
	sync.Mutex
	bot   *tgbotapi.BotAPI
	chats map[int64]struct{}
	reply tgbotapi.Message // last reply message returned
	Cache Cacher
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
			if msg.Message.Command() != "" && m.Cache != nil {
				m.processCommand(msg.Message)
			}
		} else {
			log.INFO.Printf("telegram: new chat id: %d", msg.Message.Chat.ID)
		}
		m.Unlock()
	}
}

// processCallback replies to command callbacks
func (m *Telegram) processCallback(msg *tgbotapi.CallbackQuery) {
	// log.INFO.Printf("%+v", msg)

	cb := tgbotapi.NewCallback(msg.ID, "done")
	if _, err := m.bot.AnswerCallbackQuery(cb); err != nil {
		log.ERROR.Print(err)
	}

	mode, err := api.ChargeModeString(msg.Data)
	if err != nil {
		log.ERROR.Printf("telegram: invalid callback request %v", msg)
		return
	}

	text := fmt.Sprintf("Changed mode to %s", mode)
	edit := tgbotapi.NewEditMessageText(m.reply.Chat.ID, m.reply.MessageID, text)
	if _, err := m.bot.Send(edit); err != nil {
		log.ERROR.Print(err)
	}
}

// processCommand replies to updates received
func (m *Telegram) processCommand(msg *tgbotapi.Message) {
	log.DEBUG.Printf("telegram: recv command %+v", msg.Command())

	var reply tgbotapi.MessageConfig
	switch msg.Command() {
	case cmdMode:
		buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(api.ChargeModeValues()))
		for _, m := range api.ChargeModeValues() {
			s := strings.Title(m.String())
			if m == api.PV {
				s = strings.ToTitle(s)
			}
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(s, m.String()))
		}

		reply = tgbotapi.NewMessage(msg.Chat.ID, "Select mode:")
		ikm := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))
		reply.ReplyMarkup = ikm
	case cmdStatus:
		reply = tgbotapi.NewMessage(msg.Chat.ID, "not implemented")
	default:
		log.ERROR.Printf("telegram: invalid command %s", msg.Text)
		return
	}

	if reply, err := m.bot.Send(reply); err != nil {
		log.ERROR.Print(err)
	} else {
		m.reply = reply
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
