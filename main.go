package main

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/enescakir/emoji"
	"github.com/joho/godotenv"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/sashabaranov/go-openai"
	"github.com/tiktoken-go/tokenizer"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

const gpt4MaxTokens = 8000
const gpt432MaxTokens = 32000
const greeting = "The friendly GPT-4 reviewer"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide PR number")
		return
	}

	prId := os.Args[1]

	if _, err := os.Stat(".env"); !os.IsNotExist(err) {
		err := godotenv.Load()
		check(err)
	}

	bitbucketAuth := bitbucket.NewBasicAuth(os.Getenv("BITBUCKET_USERNAME"),
		os.Getenv("BITBUCKET_APP_PASSWORD"))

	comments, err := bitbucketAuth.Repositories.PullRequests.GetComments(&bitbucket.PullRequestsOptions{
		ID:       prId,
		Owner:    "legal-i",
		RepoSlug: "legal-i",
	})
	check(err)

	commentsJson := comments.(map[string]any)
	commentsValues := commentsJson["values"].([]any)
	for _, c := range commentsValues {
		content := c.(map[string]any)["content"].(map[string]any)["html"].(string)
		// don't review a second time
		if strings.Contains(content, greeting) {
			return
		}
	}

	diff, err := bitbucketAuth.Repositories.PullRequests.Diff(&bitbucket.PullRequestsOptions{
		ID:       prId,
		Owner:    "legal-i",
		RepoSlug: "legal-i",
	})
	check(err)

	diffReader := diff.(io.ReadCloser)
	defer diffReader.Close()

	diffBytes, err := io.ReadAll(diffReader)
	check(err)
	diffString := string(diffBytes)

	enc, err := tokenizer.ForModel(tokenizer.GPT4)
	check(err)

	ids, _, err := enc.Encode(diffString)
	review := ""

	config := openai.DefaultAzureConfig(os.Getenv("AZURE_OPENAI_API_KEY"),
		os.Getenv("AZURE_OPENAI_BASE_URL"))
	openAiClient := openai.NewClientWithConfig(config)

	if len(ids) > gpt432MaxTokens {
		fmt.Println("Too many tokens")
		return
	} else {
		review, err = retry.DoWithData(
			func() (string, error) {
				return chatGptReview(openAiClient, diffString, len(ids) > gpt4MaxTokens)
			},
			retry.Context(context.Background()),
			retry.Attempts(5),
			retry.MaxDelay(1*time.Minute),
			retry.DelayType(retry.BackOffDelay),
		)
		check(err)
	}

	if review == "" {
		fmt.Println("No review")
		return
	}

	review = fmt.Sprintf(greeting+" %v:\n\n", emoji.Robot) + review

	// add comment to PR
	_, err = bitbucketAuth.Repositories.PullRequests.AddComment(&bitbucket.PullRequestCommentOptions{
		PullRequestID: prId,
		Owner:         "legal-i",
		RepoSlug:      "legal-i",
		Content:       review,
	})
	check(err)
}

func chatGptReview(openAiClient *openai.Client, diff string, gpt32 bool) (string, error) {
	prompt := `Review PR enclosed triple backticks. 
             Bullet points: -Summary, -Suggestions, -Possible bugs, -Possible performance improvements.
             Summary short with bullet points. Return Bitbucket comment.`

	model := openai.GPT4
	if gpt32 {
		model = openai.GPT432K
	}

	resp, err := openAiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       model,
			Temperature: math.SmallestNonzeroFloat32,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt + " ```" + diff + "```",
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, err
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
