package util

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type AIAgent struct {
	Type, Token, Context, Prompt, Language string
}

func AIFormatter(ai AIAgent, msg string) (string, error) {
	client := openai.NewClient(ai.Token)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					Content: "Context: " + ai.Context + "\n\n" +
						"evcc event message: " + msg + "\n\n" +
						ai.Prompt + "\n\n" +
						"Use the language of the following language iso code: " + ai.Language + "\n\n",
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
