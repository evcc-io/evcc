package util

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type AIAgent struct {
	Type, Token, Context, Prompt, Language string
}

type AIInstance struct {
	Client *openai.Client
	Ctx    context.Context
	AIAgent
}

func (i *AIInstance) NewAIAgent(p AIAgent) {
	i.Client = openai.NewClient(p.Token)
	i.Ctx = context.Background()
	i.AIAgent = p
}

func (i *AIInstance) AIFormat(msg string) (string, error) {
	resp, err := i.Client.CreateChatCompletion(
		i.Ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					Content: "Context: " + i.Context + "\n\n" +
						"evcc event message: " + msg + "\n\n" +
						i.Prompt + "\n\n" +
						"Use the language of the following language iso code: " + i.Language + "\n\n",
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return msg, nil
	}

	return resp.Choices[0].Message.Content, nil
}
