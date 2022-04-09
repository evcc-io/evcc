package push

import (
	"errors"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Telegram implements the Telegram messenger
type Telegram struct {
	sync.Mutex
	bot   *tgbotapi.BotAPI
	chats map[int64]struct{}
}

type telegramConfig struct {
	Token string
	Chats []int64
}

func init() {
	if err := tgbotapi.SetLogger(log.ERROR); err != nil {
		log.Error("telegram: %v", err)
	}
}

// NewTelegramMessenger creates new pushover messenger
func NewTelegramMessenger(token string, chats []int64) (*Telegram, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.New("telegram: invalid bot token")
	}

	m := &Telegram{
		bot:   bot,
		chats: make(map[int64]struct{}),
	}

	for _, chat := range chats {
		m.chats[chat] = struct{}{}
	}

	go m.trackChats()

	return m, nil
}

// trackChats captures ids of all chats that bot participates in
func (m *Telegram) trackChats() {
	conf := tgbotapi.NewUpdate(0)
	conf.Timeout = 1000

	for update := range m.bot.GetUpdatesChan(conf) {
		m.Lock()
		if _, ok := m.chats[update.Message.Chat.ID]; !ok {
			log.Info("telegram: new chat id: %d", update.Message.Chat.ID)
			// m.chats[update.Message.Chat.ID] = struct{}{}
		}
		m.Unlock()
	}
}

// Send sends to all receivers
func (m *Telegram) Send(title, msg string) {
	m.Lock()
	for chat := range m.chats {
		log.Debug("telegram: sending to %d", chat)

		msg := tgbotapi.NewMessage(chat, msg)
		if _, err := m.bot.Send(msg); err != nil {
			log.Error("%v", err)
		}
	}
	m.Unlock()
}
