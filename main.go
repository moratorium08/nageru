package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	flags "github.com/jessevdk/go-flags"
	"github.com/mholt/archiver"
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
	configDir     = ".config/nageru"
	configFile    = "config.toml"
	tmpDir        = "/tmp"
	tmpFilePrefix = ""
)

type Config struct {
	SlackToken string
	Channels   []string
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func IsDir(path string) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return st.IsDir(), nil
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

func genRandomFileName() (string, error) {
	// randomに16byteを受け取って文字列にする
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return tmpFilePrefix + hex.EncodeToString(b), nil
}

func stdinToTmp(filename string) error {
	reader := bufio.NewReader(os.Stdin)
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	buffer := make([]byte, 4096)

	eof := false
	for !eof {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				return err
			}
		}

		m := 0
		for m < n {
			tmp := buffer[m:n]
			d, err := writer.Write(tmp)
			if err != nil {
				return err
			}
			m += d
		}
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	return nil
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
		filename, err := genRandomFileName()
		if err != nil {
			fmt.Fprintf(os.Stderr, "乱数の取得に失敗しました\n 理由: %#v", err)
			os.Exit(-1)
		}
		// この時点では~/.config/nageru/が存在することは仮定している
		path := path.Join(tmpDir, filename)
		opts.Args.File = path
		if err := stdinToTmp(path); err != nil {
			fmt.Fprintf(os.Stderr, "入力の取得中にエラーが起きました\n 理由: %#v", err)
			os.Exit(-1)
		}
	}

	dirFlag, err := IsDir(opts.Args.File)
	if err != nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "対象となるファイルが存在しませんでした\n 理由: %#v", err)
			os.Exit(-1)
		}
	}
	if dirFlag {
		zipFilename, err := genRandomFileName()
		if err != nil {
			fmt.Fprintf(os.Stderr, "乱数の取得に失敗しました\n 理由: %#v", err)
			os.Exit(-1)
		}
		zipFilename = path.Base(opts.Args.File) + "-" + zipFilename + ".zip"

		err = archiver.Zip.Make(zipFilename, []string{opts.Args.File})
		if err != nil {
			fmt.Fprintf(os.Stderr, "フォルダーの圧縮に失敗しました\n 理由: %#v", err)
			os.Exit(-1)
		}
		opts.Args.File = zipFilename
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
