/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	SlackAPITypeWebhook = "webhook"
	SlackAPITypeDummy   = "dummy"
	CommandNameReport   = "report"
	CommandNameSend     = "send"
)

var version string

func getVersion() string {
	if version != "" {
		return version
	}

	i, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	return i.Main.Version
}

type Config struct {
	SlackAPIType string `yaml:"slackApiType"`
	WebhookURL   string `yaml:"webhookUrl"`
	Username     string `yaml:"username"`
	Verbose      bool   `yaml:"verbose"`
	Action       string `yaml:"action"`
}

func (c *Config) send(str string) error {
	if c.Verbose {
		fmt.Println(str)
	}

	if c.SlackAPIType == SlackAPITypeDummy {
		return nil
	}

	if c.Username == "" {
		if hostname, err := os.Hostname(); err != nil {
			c.Username = "unknown"
		} else {
			c.Username = hostname
		}
	}

	msg := slack.WebhookMessage{
		Username:        c.Username,
		IconEmoji:       "",
		IconURL:         "",
		Channel:         "",
		ThreadTimestamp: "",
		Text:            str,
		Attachments:     nil,
		Parse:           "",
		Blocks:          nil,
		ResponseType:    "",
		ReplaceOriginal: false,
		DeleteOriginal:  false,
		ReplyBroadcast:  false,
	}

	err := slack.PostWebhook(c.WebhookURL, &msg)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

type Executor struct {
	rootCmd *cobra.Command

	ConfigPath string
	Config     Config
}

func NewExecutor() *Executor {
	exec := &Executor{}
	exec.initRoot()
	exec.initSubcommands()

	cobra.OnInitialize(exec.readConfig)

	return exec
}

func (e *Executor) Execute() error {
	return e.rootCmd.Execute() //nolint:wrapcheck
}

func (e *Executor) initRoot() {
	//nolint:gosmopolitan
	e.rootCmd = &cobra.Command{
		Use:   "beacon-emitter",
		Short: "Slack に通知を行うツール",
		Long: `Slack に通知を行うツールです。
以下の場合の通知をサポートします。
- 引数の内容を通知
- 標準入力から受けた内容を通知
`,
		Run: nil,
	}

	e.rootCmd.PersistentFlags().StringVarP(&e.ConfigPath, "config", "c", "", "config file path")
	e.rootCmd.PersistentFlags().StringVarP(&e.Config.SlackAPIType, "apiType", "",
		"", "slack api type")
	e.rootCmd.PersistentFlags().StringVarP(&e.Config.WebhookURL, "webhookUrl", "", "", "slack webhook url")
	e.rootCmd.PersistentFlags().StringVarP(&e.Config.Username, "username", "u", "", "slack webhook username")
	e.rootCmd.PersistentFlags().BoolVarP(&e.Config.Verbose, "verbose", "v", false, "verbose output")
}

// CLI init function.
func (e *Executor) initSubcommands() {
	version := &cobra.Command{
		Use:   "version",
		Short: "print version information",
		Run:   e.Version,
	}

	e.rootCmd.AddCommand(version)

	send := &cobra.Command{
		Use:   "send [args]",
		Short: "send message",
		Run:   e.send,
	}

	e.rootCmd.AddCommand(send)

	report := &cobra.Command{
		Use:   "report ",
		Short: "report message",
		Run:   e.Report,
	}
	e.rootCmd.AddCommand(report)

	batch := &cobra.Command{
		Use:   "batch ",
		Short: "batch mode",
		Run:   e.Batch,
	}
	e.rootCmd.AddCommand(batch)
}

//nolint:cyclop
func (e *Executor) readConfig() {
	if e.ConfigPath == "" {
		return
	}

	// check file exists
	if _, err := os.Stat(e.ConfigPath); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)

		return
	}

	file, err := os.Open(e.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)

		return
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	}(file)

	// read config
	conf, err := io.ReadAll(file)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1) //nolint: gocritic

		return
	}

	var slackConf Config
	if err := yaml.Unmarshal(conf, &slackConf); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)

		return
	}

	if e.Config.SlackAPIType != "" {
		slackConf.SlackAPIType = e.Config.SlackAPIType
	}

	if e.Config.WebhookURL == "" && slackConf.WebhookURL == "" {
		e.Config.WebhookURL = SlackAPITypeWebhook
	}

	if e.Config.WebhookURL != "" {
		slackConf.WebhookURL = e.Config.WebhookURL
	}

	if e.Config.Username != "" {
		slackConf.Username = e.Config.Username
	}

	slackConf.Verbose = e.Config.Verbose

	e.Config = slackConf
}

func (e *Executor) send(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Println(args)

		if err := e.Config.send(strings.Join(args, "\n")); err != nil {
			cmd.PrintErrln(err)

			os.Exit(1)

			return
		}

		return
	}

	buf := make([]byte, 1024)

	var text string

	stdin := bufio.NewReader(os.Stdin)

	for {
		n, err := stdin.Read(buf)
		if n != 0 {
			line := string(buf[:n])
			fmt.Print(line)
			text += line
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "failed executing command with error %v\n", err)

			break
		}
	}

	if err := e.Config.send(text); err != nil {
		cmd.PrintErrln(err)

		os.Exit(1)
	}
}

func (e *Executor) Report(cmd *cobra.Command, _ []string) {
	nics, err := net.Interfaces()
	if err != nil {
		cmd.PrintErrln(err)

		return
	}

	result := make(map[string][]string)

	for _, nic := range nics {
		if nic.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := nic.Addrs()
		if err != nil {
			cmd.PrintErrln(err)

			return
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				result[nic.Name] = append(result[nic.Name], ipnet.String())
			}
		}
	}

	yml, err := yaml.Marshal(result)
	if err != nil {
		cmd.PrintErrln(err)

		return
	}

	if err := e.Config.send(string(yml)); err != nil {
		cmd.PrintErrln(err)

		return
	}
}

func (e *Executor) Batch(cmd *cobra.Command, _ []string) {
	switch e.Config.Action {
	case "":
		cmd.PrintErrln("action is not set")
		os.Exit(1)

	case CommandNameReport:
		e.Report(cmd, nil)

		return

	case CommandNameSend:
		e.send(cmd, nil)

		return

	default:
		cmd.PrintErrln("unknown action")
		os.Exit(1)
	}
}

func (e *Executor) Version(_ *cobra.Command, _ []string) {
	fmt.Println(getVersion())
}

func main() {
	ex := NewExecutor()
	if err := ex.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed executing command with error %v\n", err)
		os.Exit(1)
	}
}
