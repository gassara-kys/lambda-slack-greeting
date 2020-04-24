package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	lambda.Start(handler)
}

type slackConf struct {
	Token       string            `required:"true"`
	VToken      string            `required:"true" split_words:"true"`
	BotName     string            `required:"true" split_words:"true"`
	ChannelID   string            `required:"true" split_words:"true"`
	GreetingMap map[string]string `required:"true" split_words:"true"` // key=keyword, value=reaction_emoji
}

var (
	infoLogger  = log.New(os.Stdout, "INFO ", log.Llongfile)
	errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
)

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var conf slackConf
	err := envconfig.Process("slack", &conf)
	if err != nil {
		return serverError(err)
	}

	// parse request
	eventsAPIEvent, err := slackevents.ParseEvent(
		json.RawMessage(event.Body),
		slackevents.OptionVerifyToken(
			&slackevents.TokenComparator{
				VerificationToken: conf.VToken,
			},
		),
	)
	if err != nil {
		return serverError(err)
	}

	// Handle URL Verification
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(event.Body), &r)
		if err != nil {
			return serverError(err)
		}
		infoLogger.Println("URLVerification")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "text",
			},
			Body: r.Challenge,
		}, nil
	}

	// Handle greeting message
	var api = slack.New(conf.Token)
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			api.AddReaction("woman-raising-hand", slack.ItemRef{
				Channel:   ev.Channel,
				Timestamp: ev.TimeStamp,
			})
		case *slackevents.MessageEvent:
			if ev.User == conf.BotName ||
				ev.ChannelType != "channel" ||
				ev.Channel != conf.ChannelID {
				break
			}
			for keyword, emoji := range conf.GreetingMap {
				if strings.Contains(ev.Text, keyword) {
					// infoLogger.Printf("`%s` exists!", keyword)
					api.AddReaction(emoji, slack.ItemRef{
						Channel:   ev.Channel,
						Timestamp: ev.TimeStamp,
					})
					break
				}
			}
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "ok",
	}, nil
}

// 5xx error
func serverError(err error) (events.APIGatewayProxyResponse, error) {
	errorLogger.Println(err.Error())
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

// 4xx error
func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func parse(text string) []string {
	r := regexp.MustCompile(`\S+\+\+\s`)
	names := r.FindAllString(text, -1)
	return names
}
