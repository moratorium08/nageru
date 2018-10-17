package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/BurntSushi/toml"
	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Config string `short:"c" long:"config" description:"load config file(toml)" value-name:"CONFIG"`
	File   string `positional-args:"yes" required:"yes"`
}

const (
	configDir  = ".nageru"
	configFile = "config.toml"
)

type Config struct {
	SlackWebHook string
}

func getConfigFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := path.Join(usr.HomeDir, configDir)
	err = os.Mkdir(dir, 0755)
	if err != nil {
		return "", err
	}
	dst := path.Join(dir, configFile)
	return dst, nil
}

//  指定されたtomlファイルを~/.nageru/config.toml にコピーする
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

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}

	if opts.Config != "" {
		if err := LoadConfig(opts.Config); err != nil {
			fmt.Fprintf(os.Stderr, "Configファイルの出力に失敗しました\n 理由: %#v", err)
			os.Exit(-1)
		}
	}

}
