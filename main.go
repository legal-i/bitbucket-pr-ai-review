package main

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/enescakir/emoji"
	"github.com/joho/godotenv"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/tiktoken-go/tokenizer"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

const gpt4MaxTokens = 8000
const gpt432MaxTokens = 32000
const greeting = "The friendly GPT-4 reviewer"

const prompt = `Review Pull Request enclosed in triple backticks. Take title and description into account. Don't repeat title and description in the review.
             Output concise with bullet points (**Summary**, **Suggestions**, **Potential bugs**, **Potential performance improvements**) in Markdown format.`

type prInfo struct {
	title       string
	description string
	diff        string
}

func main() {
	prId := os.Getenv("BITBUCKET_PR_ID")
	if prId == "" {
		if len(os.Args) < 2 {
			fmt.Println("Please provide PR number")
			return
		}
		prId = os.Args[1]
	}

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
		content := c.(map[string]any)["content"].(map[string]any)
		html := content["html"].(string)
		// don't review a second time
		if strings.Contains(html, greeting) {
			fmt.Println("Already reviewed")
			return
		}
	}

	// fetch title and description of the PR
	pr, err := bitbucketAuth.Repositories.PullRequests.Get(&bitbucket.PullRequestsOptions{
		ID:       prId,
		Owner:    "legal-i",
		RepoSlug: "legal-i",
	})
	check(err)
	title := pr.(map[string]any)["title"].(string)
	description := pr.(map[string]any)["description"].(string)

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

	totalPrompt := prompt + " " + title + " " + description + " " + diffString

	ids, _, err := enc.Encode(totalPrompt)
	review := ""

	if len(ids) > gpt432MaxTokens {
		fmt.Println("Too many tokens")
		return
	} else {

		var model string
		if len(ids)+30 > gpt4MaxTokens {
			model = "gpt-4-32k"
		} else {
			model = "gpt-4"
		}

		llm, err := openai.New(
			openai.WithAPIType(openai.APITypeAzure),
			openai.WithToken(os.Getenv("AZURE_OPENAI_API_KEY")),
			openai.WithModel(model),
			openai.WithBaseURL(os.Getenv("AZURE_OPENAI_BASE_URL")),
			openai.WithEmbeddingModel("text-embedding-ada-002"),
		)
		check(err)

		prInfo := prInfo{
			title:       title,
			description: description,
			diff:        diffString,
		}

		review, err = retry.DoWithData(
			func() (string, error) {
				return chatGptReview(llm, prInfo)
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

func chatGptReview(llm *openai.LLM, prInfo prInfo) (string, error) {
	p := prompt + " ```Title: " + prInfo.title + "\nDescription: " + prInfo.description + "\nGit Diff:\n\n" + prInfo.diff + "```"
	tokenLen := llm.GetNumTokens(p)
	var maxTokens int
	if tokenLen > gpt4MaxTokens {
		maxTokens = gpt432MaxTokens - tokenLen
	} else {
		maxTokens = gpt4MaxTokens - tokenLen
	}
	completion, err := llm.Call(context.Background(), p, llms.WithTemperature(math.SmallestNonzeroFloat32), llms.WithMaxTokens(maxTokens))
	return completion, err
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
