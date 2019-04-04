package cmdspy

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
	"github.com/mgutz/ansi"
	"io"
	"reflect"
	"strings"
)

const (
	// ExitOK for exit code
	ExitOK int = 0

	// ExitErr for exit code
	ExitErr int = 1
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
	Mentions []string
	Interval int
}

// RunCLI runs as cli
func RunCLI(env Env) int {
	cli := &cli{env: env, Interval: 1800}
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

	if !Spy(args, config, c.Interval) {
		return ExitErr
	}

	return ExitOK
}
