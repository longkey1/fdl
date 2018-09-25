package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/BurntSushi/toml"
)

const (
	version = "0.1.2"
)

// Config ...
type Config struct {
	General GeneralConfig `toml:"general"`
	Slack   SlackConfig   `toml:"slack"`
}

// GeneralConfig ...
type GeneralConfig struct {
	Port         string `toml:"port"`
	DownloadPath string `toml:"download_path"`
}

// SlackConfig ...
type SlackConfig struct {
	HookURL   string `toml:"hook_url"`
	Channel   string `toml:"channel"`
	Username  string `toml:"username"`
	IconEmoji string `toml:"icon_emoji"`
}

// SlackMessage ...
type SlackMessage struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
}

// SlackWriter ...
type SlackWriter struct {
}

// Write ...
func (w SlackWriter) Write(p []byte) (int, error) {
	var text string
	text = string(p[:])
	jb, _ := json.Marshal(SlackMessage{
		Channel:   config.Slack.Channel,
		Username:  config.Slack.Username,
		Text:      text,
		IconEmoji: config.Slack.IconEmoji,
	})
	if _, err := http.Post(config.Slack.HookURL, "application/json", bytes.NewReader(jb)); err != nil {
		return 0, err
	}

	return len(text), nil
}

var config Config

func init() {
	checkVersion()
	loadConfig()

	w := new(SlackWriter)
	log.SetOutput(w)

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/register", registerHandler)
}

func checkVersion() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("fdl version %s\n", version)
		os.Exit(0)
	}
}

func loadConfig() {
	var path string
	flag.StringVar(&path, "c", "config.tml", "configuration file path")
	flag.Parse()

	if _, err := toml.DecodeFile(path, &config); err != nil {
		panic(err)
	}
}

func main() {
	log.Println("start server")
	http.ListenAndServe(":"+config.General.Port, nil)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.New("index").Parse(TemplateIndex)
	if err != nil {
		panic(err)
	}
	if err := tpl.Execute(w, map[string]string{
		"Src": r.FormValue("src"),
		"Dst": r.FormValue("dst"),
	}); err != nil {
		panic(err)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	go download(r.FormValue("src"), r.FormValue("dst"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func download(src string, dst string) {
	fn := getOutputFilename(src, dst)
	log.Printf("download start: %s", fn)

	resp, err := http.Get(src)
	if err != nil {
		log.Println(err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	file, err := os.OpenFile(filepath.Join(config.General.DownloadPath, fn), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
	}
	defer func() {
		file.Close()
		log.Printf("download end: %s", fn)
	}()

	file.Write(body)
}

func getOutputFilename(src string, dst string) string {
	u, err := url.Parse(src)
	if err != nil {
		panic(err)
	}
	fn := u.Path
	if len(dst) > 0 {
		fn = dst + filepath.Ext(fn)
	}

	return fn
}

const TemplateIndex = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>fdl</title>
    <link rel="stylesheet" href="//maxcdn.bootstrapcdn.com/bootswatch/3.3.6/slate/bootstrap.min.css">
    <link rel="shortcut icon" href="//cloud.githubusercontent.com/assets/58566/24833932/692bab5c-1d12-11e7-999b-ec96b97a5721.png">
  </head>
  <body>
    <div class="container">
      <h1>fdl</h1>
      <form action="/register" method="get" role="form">
        <div class="form-group">
          <label for="inputdefault">SOURCE URL</label>
          <input class="form-control" id="inputdefault" type="text" name="src" value="{{.Src}}">
        </div>
        <div class="form-group">
          <label for="inputdefault">DESTINATION FILE NAME</label>
          <input class="form-control" id="inputdefault" type="text" name="dst" value="{{.Dst}}">
        </div>
        <button type="submit" class="btn btn-primary">Submit</button>
      </form>
    </div>
    <script src="//code.jquery.com/jquery-3.0.0.min.js" integrity="sha256-JmvOoLtYsmqlsWxa7mDSLMwa6dZ9rrIdtrrVYRnDRH0=" crossorigin="anonymous"></script>
  </body>
</html>
`
