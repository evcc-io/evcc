package push

import (
	"errors"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func init() {
	registry.Add("telegram", NewTelegramFromConfig)
}

// Telegram implements the Telegram messenger
type Telegram struct {
	log *util.Logger
	sync.Mutex
	bot   *tgbotapi.BotAPI
	chats map[int64]struct{}
}

// NewTelegramFromConfig creates new pushover messenger
func NewTelegramFromConfig(other map[string]interface{}) (Messenger, error) {
	var cc struct {
		Token string
		Chats []int64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	bot, err := tgbotapi.NewBotAPI(cc.Token)
	if err != nil {
		return nil, errors.New("telegram: invalid bot token")
	}

	log := util.NewLogger("telegram").Redact(cc.Token)
	_ = tgbotapi.SetLogger(log.ERROR)

	for _, i := range cc.Chats {
		log.Redact(strconv.FormatInt(i, 10))
	}

	m := &Telegram{
		log:   log,
		bot:   bot,
		chats: make(map[int64]struct{}),
	}

	for _, chat := range cc.Chats {
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
			m.log.INFO.Printf("new chat id: %d", update.Message.Chat.ID)
		}
		m.Unlock()
	}
}

// Send sends to all receivers
func (m *Telegram) Send(title, msg string) {
	m.Lock()
	for chat := range m.chats {
		m.log.DEBUG.Printf("sending to %d", chat)

		msg := tgbotapi.NewMessage(chat, msg)
		if _, err := m.bot.Send(msg); err != nil {
			m.log.ERROR.Println("send:", err)
		}
	}
	m.Unlock()
}
