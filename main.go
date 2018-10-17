package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	flags "github.com/jessevdk/go-flags"
	"github.com/nlopes/slack"
)

var opts struct {
	Config  string `long:"config" description:"load config file(toml)" value-name:"CONFIG"`
	Message string `short:"m" long:"message" description:"comment attached to the file" value-name:"MESSAGE"`
	Channel string `short:"c" long:"channel" description:"channel in which your file will be sent" value-name:"CHANNEL"`
	Title   string `short:"t" long:"title" description:"title attached to the file" value-name:"TITLE"`
	Args    struct {
		File string
	} `positional-args:"yes"`
}

const (
	configDir  = ".config/nageru"
	configFile = "config.toml"
)

type Config struct {
	SlackToken string
	Channels   []string
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func getConfigFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := path.Join(usr.HomeDir, configDir)
	if !Exists(dir) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return "", err
		}
	}
	dst := path.Join(dir, configFile)
	return dst, nil
}

func saveConfig(config Config) error {
	dst, err := getConfigFilePath()
	w, err := os.Create(dst)
	defer w.Close()

	if err != nil {
		return err
	}
	e := toml.NewEncoder(w)
	return e.Encode(config)
}

// LoadConfig は、指定されたtomlファイルを~/.nageru/config.toml にコピーする
func LoadConfig(file string) error {
	dst, err := getConfigFilePath()
	if err != nil {
		return err
	}

	// validなtomlファイルかどうかを確認する
	var config Config
	_, err = toml.DecodeFile(file, &config)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReadConfig() (*Config, error) {
	src, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}
	var config Config
	if !Exists(src) {
		fmt.Printf("Configファイルが無いようです。取り急ぎ、FileUploadが許可されたTokenがあれば作るので教えてください\nToken: ")
		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		token = strings.TrimSuffix(token, "\n")
		fmt.Printf("Channel: ")
		channel, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		channel = strings.TrimSuffix(channel, "\n")
		config = Config{token, []string{channel}}
		if err := saveConfig(config); err != nil {
			return nil, err
		}
	} else {
		_, err = toml.DecodeFile(src, &config)
		if err != nil {
			return nil, err
		}
	}
	return &config, nil
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(-1)
	}

	if opts.Config != "" {
		if err := LoadConfig(opts.Config); err != nil {
			fmt.Fprintf(os.Stderr, "Configファイルの出力に失敗しました\n 理由: %#v", err)
			os.Exit(-1)
		}
	}

	config, err := ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configファイルの読み込みに失敗しました\n 理由: %#v", err)
		os.Exit(-1)
	}

	if opts.Args.File == "" {
		return
	}

	file, err := os.Open(opts.Args.File)
	defer file.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ファイルの読み込みに失敗しました\n 理由: %#v", err)
		os.Exit(-1)
	}

	var channels []string

	if opts.Channel != "" {
		channels = []string{opts.Channel}
	} else {
		channels = config.Channels
	}

	params := slack.FileUploadParameters{
		Channels:       channels,
		Filename:       path.Base(opts.Args.File),
		InitialComment: opts.Message,
		Reader:         file,
		Title:          opts.Title,
	}

	api := slack.New(config.SlackToken)
	if _, err := api.UploadFile(params); err != nil {
		fmt.Fprintf(os.Stderr, "ファイルのアップロードに失敗しました\n 理由: %#v", err)
		os.Exit(-1)
	}
}
