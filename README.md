# nageru

Slackに雑にファイルを投げるツール

## インストール

```
$ go get -u github.com/moratorium08/nageru
```

または、バイナリを [release](https://github.com/moratorium08/nageru/releases) から落としてください

## 使い方

### 例

```
$ ls
cookies
$ nageru cookies -m "this is message(not required)" -c random -t "this is title(not required)"
```

より詳しくは以下のようなオプションが使えます。

```
Usage:
  nageru [OPTIONS] File

Application Options:
      --config=CONFIG      load config file(toml)
  -m, --message=MESSAGE    comment attached to the file
  -c, --channel=CHANNEL    channel in which your file will be sent
  -t, --title=TITLE        title attached to the file

Help Options:
  -h, --help               Show this help message
```

## Slack Tokenの設定

重要なのはslackのConfigを適切に設定しないと動かない点です。これは、初回起動時に設定ができます。API Tokenの正しい取り方は、[Qiita](https://qiita.com/ykhirao/items/3b19ee6a1458cfb4ba21) を見てください。


## tomlファイル

ところでtomlファイルを書いて設定することもできます。これは、repoのconfig.tomlを参考にしてください。SlackTokenのところを適切に設定した後に

```
$ nageru --config config.toml
```

でファイルが適切に設定されると思います

## その他機能

### stdinから送信

ファイル名を指定しなかったときは、stdinからデータを取得し、EOFを受け取ったらそれまでに受け取ったデータを送信します

### ディレクトリを送信

指定されたファイルがディレクトリだった場合は、そのディレクトリを圧縮して送信します

