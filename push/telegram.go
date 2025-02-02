package push

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func init() {
	registry.AddCtx("telegram", NewTelegramFromConfig)
}

// Telegram implements the Telegram messenger
type Telegram struct {
	log *util.Logger
	sync.Mutex
	bot   *bot.Bot
	chats map[int64]struct{}
}

// NewTelegramFromConfig creates new pushover messenger
func NewTelegramFromConfig(ctx context.Context, other map[string]interface{}) (Messenger, error) {
	var cc struct {
		Token string
		Chats []int64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("telegram").Redact(cc.Token)

	m := &Telegram{
		log:   log,
		chats: make(map[int64]struct{}),
	}

	bot, err := bot.New(cc.Token, bot.WithDefaultHandler(m.handler), bot.WithErrorsHandler(func(err error) {
		log.ERROR.Println(err)
	}), bot.WithDebugHandler(func(format string, args ...interface{}) {
		log.TRACE.Printf(format, args...)
	}))
	if err != nil {
		return nil, errors.New("invalid bot token")
	}

	m.bot = bot

	go bot.Start(ctx)

	for _, chat := range cc.Chats {
		log.Redact(strconv.FormatInt(chat, 10))
		m.chats[chat] = struct{}{}
	}

	return m, nil
}

// handler captures ids of all chats that bot participates in
func (m *Telegram) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	m.Lock()
	defer m.Unlock()

	if _, ok := m.chats[update.Message.Chat.ID]; !ok {
		m.log.INFO.Printf("new chat id: %d", update.Message.Chat.ID)
	}
}

// Send sends to all receivers
func (m *Telegram) Send(title, msg string) {
	m.Lock()
	defer m.Unlock()

	for chat := range m.chats {
		m.log.DEBUG.Printf("sending to %d", chat)

		if _, err := m.bot.SendMessage(context.Background(), &bot.SendMessageParams{
			ChatID: chat,
			Text:   msg,
		}); err != nil {
			m.log.ERROR.Println("send:", err)
		}
	}
}
