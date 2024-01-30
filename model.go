package main

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/Dannystu12/go-notifier"
	"github.com/getlantern/systray"
)

type Jobs struct {
	list   []*Job
	menus  []*systray.MenuItem
	mu     sync.Mutex
	cancel func()
	wg     sync.WaitGroup

	loaded atomic.Bool
}

func NewJobs() *Jobs {
	return &Jobs{
		cancel: func() {},
	}
}

func (js *Jobs) Setup(newList []*Job) (menus []*systray.MenuItem) {
	js.mu.Lock()
	defer js.mu.Unlock()
	defer n.DeliverNotification(notifier.Notification{
		Title:   "Gortcut",
		Message: "Succesfully setup!",
	})
	js.loaded.Store(true)

	js.cancel()
	js.wg.Wait()

	for _, menu := range js.menus {
		menu.Disable()
	}

	newLen := len(newList)
	for len(js.menus) < newLen+2 {
		js.menus = append(js.menus, systray.AddMenuItem("", ""))
	}
	for i := len(js.menus) - 1; i > newLen+1; i-- {
		js.menus[i].Hide()
	}

	ctx, cancel := context.WithCancel(context.Background())
	js.cancel = cancel
	js.wg.Add(newLen + 2)
	for i, j := range newList {
		js.menus[i].SetTitle(j.Title)
		js.menus[i].Enable()
		go j.Run(ctx, &js.wg, js.menus[i])
	}

	{
		js.menus[newLen].SetTitle("Reload")
		js.menus[newLen].Enable()
		go func(wg *sync.WaitGroup, menu *systray.MenuItem) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case <-menu.ClickedCh:
				go Load()
			}
		}(&js.wg, js.menus[newLen])
	}

	{
		js.menus[newLen+1].SetTitle("Quit")
		js.menus[newLen+1].Enable()
		go func(wg *sync.WaitGroup, menu *systray.MenuItem) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case <-menu.ClickedCh:
				systray.Quit()
			}
		}(&js.wg, js.menus[newLen+1])
	}

	return menus
}

type StateName string

type Job struct {
	Title  string
	Start  StateName
	States map[StateName]*State
}

func (j *Job) Run(ctx context.Context, wg *sync.WaitGroup, menu *systray.MenuItem) {
	defer wg.Done()
	st := j.Start
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		st = j.States[st].Run(ctx, menu)
	}
}

type State struct {
	Name    StateName
	Title   string
	Cmd     []string
	AutoRun bool
	Match   []*Match
}

func (s *State) Run(ctx context.Context, menu *systray.MenuItem) StateName {
	menu.SetTitle(s.Title)
	if !s.AutoRun {
		select {
		case <-ctx.Done():
			return ""
		case <-menu.ClickedCh:
		}
	}
	menu.Disable()
	defer menu.Enable()
	cmd := &Cmd{
		osCmd: exec.CommandContext(ctx, s.Cmd[0], s.Cmd[1:]...),
	}
	cmd.osCmd.Stdout = &cmd.Stdout
	cmd.osCmd.Stderr = &cmd.Stderr
	cmd.osCmd.Run()
	select {
	case <-ctx.Done():
		return ""
	default:
	}
	for _, m := range s.Match {
		if m.match(cmd) {
			if m.Do.Show != nil {
				n.DeliverNotification(notifier.Notification{
					Title:   "Gortcut",
					Message: *m.Do.Show,
				})
			}
			if m.Do.Goto == "STAY" {
				return s.Name
			}
			return (m.Do.Goto)
		}
	}
	n.DeliverNotification(notifier.Notification{
		Title:     "Gortcut",
		Message:   "Unhandled result: state remains at " + string(s.Name),
		ImagePath: "dialog-warning",
	})
	return s.Name
}

type CaseTerm struct {
	Stdout   *string
	Stderr   *string
	ExitCode *int
	Success  *bool

	once          sync.Once
	stdoutPattern *regexp.Regexp
	stderrPattern *regexp.Regexp
}

type Cmd struct {
	osCmd  *exec.Cmd
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

func (t *CaseTerm) prepare() {
	t.once.Do(func() {
		if t.Stdout != nil {
			t.stdoutPattern = regexp.MustCompile(*t.Stdout)
		}
		if t.Stderr != nil {
			t.stderrPattern = regexp.MustCompile(*t.Stderr)
		}
	})
}

func (t *CaseTerm) match(cmd *Cmd) bool {
	t.prepare()
	if t.stdoutPattern != nil {
		if !t.stdoutPattern.Match(cmd.Stdout.Bytes()) {
			return false
		}
	}
	if t.stderrPattern != nil {
		if !t.stderrPattern.Match(cmd.Stderr.Bytes()) {
			return false
		}
	}
	if t.ExitCode != nil {
		if cmd.osCmd.ProcessState.ExitCode() != *t.ExitCode {
			return false
		}
	}
	if t.Success != nil {
		if cmd.osCmd.ProcessState.Success() != *t.Success {
			return false
		}
	}
	return true
}

type Case []*CaseTerm

func (c Case) match(cmd *Cmd) bool {
	for _, t := range c {
		if t.match(cmd) {
			return true
		}
	}
	return false
}

type Do struct {
	Show *string
	Goto StateName
}

type Match struct {
	Case Case
	Do   Do
}

func (m *Match) match(cmd *Cmd) bool {
	return m.Case.match(cmd)
}
