package openai

// ModelList is a list of models that are supported by the OpenAI adaptor.
// This is not an exhaustive list, and users can use other models by manually configuring them.
var ModelList = []string{
	"gpt-3.5-turbo",
	"gpt-3.5-turbo-0125",
	"gpt-3.5-turbo-1106",
	"gpt-3.5-turbo-instruct",
	"gpt-3.5-turbo-16k",
	"gpt-4",
	"gpt-4-0613",
	"gpt-4-32k",
	"gpt-4-32k-0613",
	"gpt-4-turbo",
	"gpt-4-turbo-preview",
	"gpt-4-turbo-2024-04-09",
	"gpt-4-1106-preview",
	"gpt-4-0125-preview",
	"gpt-4-vision-preview",
	"gpt-4o",
	"gpt-4o-2024-05-13",
	"gpt-4o-mini",
	"gpt-4o-mini-2024-07-18",
	"o1-preview",
	"o1-mini",
	"dall-e-2",
	"dall-e-3",
	"text-embedding-3-small",
	"text-embedding-3-large",
	"text-embedding-ada-002",
	"tts-1",
	"tts-1-hd",
	"whisper-1",
}

const (
	BaseURL              = "https://api.openai.com"
	ChatCompletionsURL   = "%s/v1/chat/completions"
	ImagesGenerationsURL = "%s/v1/images/generations"
)
