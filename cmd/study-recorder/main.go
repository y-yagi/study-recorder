package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/urfave/cli"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/toml"
)

type config struct {
	URL   string `toml:"url"`
	Token string `toml:"token"`
}

var (
	cfg config
)

const (
	cmd = "study-recorder"
)

type study struct {
	Content string `json:"content"`
	Hour    int    `json:"hour"`
	Minute  int    `json:"minute"`
	Theme   string `json:"theme"`
}

func main() {
	os.Exit(run(os.Args))
}

func init() {
	if configure.Exist(cmd) {
		err := configure.Load(cmd, &cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg.URL = "https://my-study.herokuapp.com/"
		cfg.Token = ""
		configure.Save(cmd, cfg)
	}
}

func run(args []string) int {
	app := cli.NewApp()
	app.Name = cmd
	app.Usage = "CLI for My Study"
	app.Version = "0.1.0"
	app.Action = appRun
	app.Commands = commands()

	return msg(app.Run(args))
}

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		return 1
	}
	return 0
}

func commands() []cli.Command {
	return []cli.Command{
		cli.Command{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "add a new study",
			Action:  addStudy,
		},
		cli.Command{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "edit config",
			Action:  editConfig,
		},
	}
}

func appRun(c *cli.Context) error {
	cli.ShowAppHelp(c)
	return nil
}

func addStudy(c *cli.Context) error {
	var s study
	if err := generateStudyReport(&s); err != nil {
		return err
	}
	postData, _ := json.Marshal(s)

	req, err := http.NewRequest("POST", cfg.URL+"/api/studies", bytes.NewBuffer(postData))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "JWT "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	fmt.Printf("result: %s\n", body)
	return nil
}

func generateStudyReport(s *study) error {
	tmpfile, err := ioutil.TempFile("", "study-report.toml")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(`theme = ""
hour = 0
minute = 0
content = """

"""`)
	if err != nil {
		return err
	}

	if err = editReport(tmpfile.Name()); err != nil {
		return err
	}

	_, err = toml.DecodeFile(tmpfile.Name(), s)
	return err
}

func editConfig(c *cli.Context) error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(cmd, editor)
}

func editReport(name string) error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	cmd := exec.Command(editor, name)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
