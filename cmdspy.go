package cmdspy

import (
	"bufio"
	"fmt"
	"github.com/mgutz/ansi"
	"os"
	"os/exec"
	"strings"
	"time"

	slacli "github.com/monochromegane/slack-incoming-webhooks"
)

const (
	OutColor = "green"

	ErrColor = "red"
)

// Slack struct
type Slack struct {
	Title      string
	Message    string
	Color      string
	IconEmoji  string
	WebhookURL string
	Channel    string
	Mentions   []string
}

// Spy
func Spy(args []string, config Config, interval int) bool {
	start := time.Now()

	emoji := ":sunglasses:"
	if config.Emoji != "" {
		emoji = config.Emoji
	}

	sl := Slack{
		WebhookURL: config.Url,
		Channel:    config.Channel,
		IconEmoji:  emoji,
		Mentions:   config.Mentions,
		Title:      args[0],
		Message:    "Exec Command",
		Color:      "#5CB589",
	}

	s := strings.Split(args[0], " ")
	cmd := exec.Command(s[0], s[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sl.Message = err.Error()
		sl.Color = "#961D13"
		sl.Message = err.Error()
		postMessageToSlack(sl, true)
		return false
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sl.Message = err.Error()
		sl.Color = "#961D13"
		sl.Message = err.Error()
		postMessageToSlack(sl, true)
		return false
	}

	if err := cmd.Start(); err != nil {
		sl.Message = err.Error()
		sl.Color = "#961D13"
		sl.Message = err.Error()
		postMessageToSlack(sl, true)
		return false
	}

	line, err := getPs(cmd.Process.Pid)
	if err != nil {
		sl.Message = fmt.Sprintf("%s\n%s", sl.Message, err.Error())
	} else {
		sl.Message = fmt.Sprintf("%s\n%s", sl.Message, line)
	}

	postMessageToSlack(sl, false)

	streamReader := func(scanner *bufio.Scanner, outputChan chan string, doneChan chan bool) {
		defer close(outputChan)
		defer close(doneChan)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		doneChan <- true
	}

	stdoutScanner := bufio.NewScanner(stdout)
	stdoutOutputChan := make(chan string)
	stdoutDoneChan := make(chan bool)
	stderrScanner := bufio.NewScanner(stderr)
	stderrOutputChan := make(chan string)
	stderrDoneChan := make(chan bool)
	go streamReader(stdoutScanner, stdoutOutputChan, stdoutDoneChan)
	go streamReader(stderrScanner, stderrOutputChan, stderrDoneChan)

	nextInterval := config.Interval
	if interval == 0 {
		nextInterval = interval
	}

	state := true
	for state {
		select {
		case <-stdoutDoneChan:
			state = false
		case line := <-stdoutOutputChan:
			fmt.Println(ansi.Color(line, OutColor))
		case line := <-stderrOutputChan:
			fmt.Println(ansi.Color(line, ErrColor))
		default:
			now := time.Now()
			duration := now.Sub(start)
			if int(duration.Seconds()) >= nextInterval {
				sl.Message = fmt.Sprintf("%s から %02d:%02d:%02d 経過", start.Format("2006-01-02 15:04:05"), int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
				line, err := getPs(cmd.Process.Pid)
				if err != nil {
					sl.Message = fmt.Sprintf("%s\n%s", sl.Message, err.Error())
				} else {
					sl.Message = fmt.Sprintf("%s\n%s", sl.Message, line)
				}
				postMessageToSlack(sl, false)
				nextInterval += interval
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		sl.Message = err.Error()
		sl.Color = "#961D13"
		postMessageToSlack(sl, true)
		return false
	} else {
		sl.Message = "Success"
		postMessageToSlack(sl, false)
	}

	return true
}

var slackReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// postMessageToSlack
func postMessageToSlack(sl Slack, withMention bool) error {
	if sl.Channel != "" && !strings.HasPrefix(sl.Channel, "#") {
		sl.Channel = "#" + sl.Channel
	}

	sl.Message = slackReplacer.Replace(sl.Message)

	cli := slacli.Client{WebhookURL: sl.WebhookURL}

	var text string
	if withMention {
		for _, mention := range sl.Mentions {
			text = text + fmt.Sprintf("%s ", mention)
		}
		if len(text) > 0 {
			text = fmt.Sprintf("%sエラーが発生しました", text)
		}
	}

	return cli.Post(&slacli.Payload{
		Username:  "cmdspy",
		IconEmoji: sl.IconEmoji,
		Channel:   sl.Channel,
		Text:      text,
		Attachments: []*slacli.Attachment{
			{
				Color:      sl.Color,
				Title:      sl.Title,
				Text:       sl.Message,
				MarkdownIn: []string{"Text"},
			},
		},
	})
}

// getPs
func getPs(pid int) (string, error) {
	ps := []string{"ps", "-p", fmt.Sprintf("%d", pid), "-o", "user,pid,%cpu,%mem,vsz,rss,tt,state,start,time"}
	c := exec.Command(ps[0], ps[1:]...)
	c.Stdin = os.Stdin
	c.Env = append(os.Environ(), "LC_ALL=POSIX")
	out, err := c.Output()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("```%s```", string(out)), nil
}
