package main

import "github.com/gox/frm/log"

func main() {
	log.LoadConfig(&log.Config{
		Url:     "http://127.0.0.1:8081",
		Project: "test",
		Mod:     "aaaa",
		User:    "admin",
		Key:     "680f87b9734f52ef0b27fe87",
	})

	log.Debug("hello world")
	log.Info("Hello world")
	log.Warn("Hello world")
	log.Error("Hello world")
	log.Fatal("Hello world")
}
