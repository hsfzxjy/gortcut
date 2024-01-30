package main

import "github.com/Dannystu12/go-notifier"

var n notifier.Notifier

func init() {
	var err error
	n, err = notifier.NewNotifier()
	if err != nil {
		panic(err)
	}
}
