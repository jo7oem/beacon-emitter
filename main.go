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
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	SlackAPITypeWebhook = "webhook"
)

type Config struct {
	SlackAPIType string `yaml:"slackApiType"`
	WebhookURL   string `yaml:"webhookUrl"`
	Username     string `yaml:"username"`
	Verbose      bool   `yaml:"verbose"`
}

func (c *Config) send(str string) error {
	if c.Username == "" {
		if hostname, err := os.Hostname(); err != nil {
			c.Username = "unknown"
		} else {
			c.Username = hostname
		}
	}

	if c.Verbose {
		fmt.Println(str)
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

	ConfigPath  string
	SlackConfig Config
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
		Run: e.echo,
	}

	e.rootCmd.PersistentFlags().StringVarP(&e.ConfigPath, "config", "c", "", "config file path")
	e.rootCmd.PersistentFlags().StringVarP(&e.SlackConfig.SlackAPIType, "apiType", "",
		SlackAPITypeWebhook, "slack api type")
	e.rootCmd.PersistentFlags().StringVarP(&e.SlackConfig.WebhookURL, "webhookUrl", "", "", "slack webhook url")
	e.rootCmd.PersistentFlags().StringVarP(&e.SlackConfig.Username, "username", "u", "", "slack webhook username")
	e.rootCmd.PersistentFlags().BoolVarP(&e.SlackConfig.Verbose, "verbose", "v", false, "verbose output")
}

// CLI init function.
func (e *Executor) initSubcommands() {
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
}

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

	if e.SlackConfig.SlackAPIType != "" {
		slackConf.SlackAPIType = e.SlackConfig.SlackAPIType
	}

	if e.SlackConfig.WebhookURL != "" {
		slackConf.WebhookURL = e.SlackConfig.WebhookURL
	}

	if e.SlackConfig.Username != "" {
		slackConf.Username = e.SlackConfig.Username
	}

	slackConf.Verbose = e.SlackConfig.Verbose

	e.SlackConfig = slackConf
}

func (e *Executor) echo(_ *cobra.Command, args []string) {
	fmt.Println(e.ConfigPath)
	fmt.Println("Args: ", args)
}

func (e *Executor) send(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Println(args)

		if err := e.SlackConfig.send(strings.Join(args, "\n")); err != nil {
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

	if err := e.SlackConfig.send(text); err != nil {
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

	if err := e.SlackConfig.send(string(yml)); err != nil {
		cmd.PrintErrln(err)

		return
	}
}

func main() {
	ex := NewExecutor()
	if err := ex.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed executing command with error %v\n", err)
		os.Exit(1)
	}
}
