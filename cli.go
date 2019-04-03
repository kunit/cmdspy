package cmdspy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
	"github.com/mgutz/ansi"
	"github.com/monochromegane/slack-incoming-webhooks"
)

const (
	// ExitOK for exit code
	ExitOK int = 0

	// ExitErr for exit code
	ExitErr int = 1
)

const (
	OutColor = "green"

	ErrColor = "red"
)

// cli struct
type cli struct {
	env      Env
	command  string
	args     []string
	Config   string `long:"config" short:"c" description:"path to configuration file"`
	Interval int    `long:"interval" arg:"seconds" short:"i" description:"report interval (default: 600)"`
	Help     bool   `long:"help" short:"h" description:"show this help message and exit"`
	Version  bool   `long:"version" short:"v" description:"prints the version number"`
}

// Env struct
type Env struct {
	Out, Err io.Writer
	Args     []string
	Version  string
}

// Config
type Config struct {
	Url      string
	Channel  string
	Emoji    string
	Interval int
}

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

// RunCLI runs as cli
func RunCLI(env Env) int {
	cli := &cli{env: env, Interval: 600}
	return cli.run()
}

// buildHelp
func (c *cli) buildHelp(names []string) []string {
	var help []string
	t := reflect.TypeOf(cli{})

	for _, name := range names {
		f, ok := t.FieldByName(name)
		if !ok {
			continue
		}

		tag := f.Tag
		if tag == "" {
			continue
		}

		var o, a string
		if a = tag.Get("arg"); a != "" {
			a = fmt.Sprintf("=%s", a)
		}
		if s := tag.Get("short"); s != "" {
			o = fmt.Sprintf("-%s, --%s%s", tag.Get("short"), tag.Get("long"), a)
		} else {
			o = fmt.Sprintf("--%s%s", tag.Get("long"), a)
		}

		desc := tag.Get("description")
		if i := strings.Index(desc, "\n"); i >= 0 {
			var buf bytes.Buffer
			buf.WriteString(desc[:i+1])
			desc = desc[i+1:]
			const indent = "                        "
			for {
				if i = strings.Index(desc, "\n"); i >= 0 {
					buf.WriteString(indent)
					buf.WriteString(desc[:i+1])
					desc = desc[i+1:]
					continue
				}
				break
			}
			if len(desc) > 0 {
				buf.WriteString(indent)
				buf.WriteString(desc)
			}
			desc = buf.String()
		}
		help = append(help, fmt.Sprintf("  %-40s %s", o, desc))
	}

	return help
}

// showHelp
func (c *cli) showHelp() {
	opts := strings.Join(c.buildHelp([]string{
		"Config",
		"Interval",
	}), "\n")

	help := `Usage: cmdspy [--version] [--help] <options> "command <arg1> <arg2>..."

Options:
%s
`
	fmt.Fprintf(c.env.Out, help, opts)
}

// run
func (c *cli) run() int {
	p := flags.NewParser(c, flags.PassDoubleDash)
	args, err := p.ParseArgs(c.env.Args)
	if err != nil || c.Help {
		c.showHelp()
		return ExitOK
	}

	if c.Version {
		fmt.Fprintf(c.env.Err, "cmdspy version %s\n", c.env.Version)
		return ExitOK
	}

	if len(c.Config) == 0 {
		fmt.Fprint(c.env.Err, ansi.Color("Error: Required config option\n\n", ErrColor))
		c.showHelp()
		return ExitErr
	}

	var config Config
	_, err = toml.DecodeFile(c.Config, &config)
	if err != nil {
		fmt.Fprint(c.env.Err, ansi.Color(fmt.Sprintf("Error: DecodeFile Failure %s\n\n%v\n\n", c.Config, err.Error()), ErrColor))
		c.showHelp()
		return ExitErr
	}

	if len(config.Url) == 0 || len(config.Channel) == 0 {
		fmt.Fprint(c.env.Err, ansi.Color("Error: Required url and channel\n\n", ErrColor))
		c.showHelp()
		return ExitErr
	}

	if len(args) == 0 {
		c.showHelp()
		return ExitErr
	}

	start := time.Now()

	sl := Slack{
		WebhookURL: config.Url,
		Channel:    config.Channel,
		IconEmoji:  config.Emoji,
		Title:      args[0],
		Message:    "Exec Command",
		Color:      "#5CB589",
	}

	PostMessageToSlack(sl)

	s := strings.Split(args[0], " ")
	cmd := exec.Command(s[0], s[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sl.Message = "Error"
		sl.Color = "#961D13"
		sl.Message = err.Error()
		PostMessageToSlack(sl)
		return ExitErr
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sl.Message = "Error"
		sl.Color = "#961D13"
		sl.Message = err.Error()
		PostMessageToSlack(sl)
		return ExitErr
	}

	if err := cmd.Start(); err != nil {
		sl.Message = "Error"
		sl.Color = "#961D13"
		sl.Message = err.Error()
		PostMessageToSlack(sl)
		return ExitErr
	}

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
	if c.Interval == 0 {
		nextInterval = c.Interval
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
				sl.Message = fmt.Sprintf("%s から %02d:%02d:%02d 経過", start.Format("2006-01-02 15:04:05"), int(duration.Hours()), int(duration.Minutes()), int(duration.Seconds()))
				PostMessageToSlack(sl)
				nextInterval += c.Interval
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		sl.Message = "Error"
		sl.Color = "#961D13"
		PostMessageToSlack(sl)
		return ExitErr
	} else {
		sl.Message = "Success"
		PostMessageToSlack(sl)
	}

	return ExitOK
}

var slackReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// PostMessageToSlack
func PostMessageToSlack(sl Slack) error {
	if sl.Channel != "" && !strings.HasPrefix(sl.Channel, "#") {
		sl.Channel = "#" + sl.Channel
	}

	sl.Message = slackReplacer.Replace(sl.Message)

	cli := slack_incoming_webhooks.Client{WebhookURL: sl.WebhookURL}

	return cli.Post(&slack_incoming_webhooks.Payload{
		Username:  "cmdspy",
		IconEmoji: sl.IconEmoji,
		Channel:   sl.Channel,
		Attachments: []*slack_incoming_webhooks.Attachment{
			{
				Color: sl.Color,
				Title: sl.Title,
				Text:  sl.Message,
			},
		},
	})
}
