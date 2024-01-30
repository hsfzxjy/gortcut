package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/Dannystu12/go-notifier"
	"github.com/getlantern/systray"
)

//go:embed app.ico
var icon []byte

var jobs = NewJobs()

func main() {
	EnsureSystemd()
	systray.SetIcon(icon)
	systray.Run(Load, func() {})
}

func Load() {
	jobList, err := ParseConfig(configFilePath)
	if err != nil {
		n.DeliverNotification(notifier.Notification{
			Title:     "Gortcut Init Error",
			Message:   err.Error(),
			ImagePath: "dialog-error",
		})
		if !jobs.loaded.Load() {
			systray.Quit()
		}
		return
	}
	jobs.Setup(jobList)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var configFilePath string

func EnsureSystemd() {
	home, err := os.UserHomeDir()
	check(err)

	configDir := path.Join(home, ".config", "gortcut")
	check(os.MkdirAll(configDir, 0755))
	configFilePath = path.Join(configDir, "config.cue")

	if _, err := os.Stat(configFilePath); err != nil {
		check(os.WriteFile(configFilePath, []byte(DefaultConfig), 0644))
	}

	if len(os.Args) > 1 && os.Args[1] == "daemonized" {
		return
	}

	exe, err := os.Executable()
	check(err)
	service := path.Join(home, ".config", "systemd", "user", "gortcut.service")

	desiredContent := fmt.Sprintf(SystemdTemplate, exe)

	var shouldWriteService bool
	if _, err := os.Stat(service); err != nil && os.IsNotExist(err) {
		shouldWriteService = true
	} else {
		data, err := os.ReadFile(service)
		check(err)
		shouldWriteService = string(data) != desiredContent
	}

	if shouldWriteService {
		check(os.WriteFile(service, []byte(desiredContent), 0644))
		check(exec.Command("systemctl", "--user", "daemon-reload").Run())
		check(exec.Command("systemctl", "--user", "enable", "gortcut").Run())
		check(exec.Command("systemctl", "--user", "start", "gortcut").Run())
	} else {
		check(exec.Command("systemctl", "--user", "start", "gortcut").Run())
	}
	os.Exit(0)
}

const DefaultConfig = `jobs: []`

const SystemdTemplate = `[Unit]
Description=Gortcut
After=network.target

[Service]
Type=simple
ExecStart=%s daemonized

[Install]
WantedBy=default.target
`
