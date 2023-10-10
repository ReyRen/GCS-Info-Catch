package main

import (
	"github.com/sevlyar/go-daemon"
	"log"
)

func main() {
	//Setup daemon system
	cntxt := &daemon.Context{
		PidFileName: "gcsInfoCatch.pid",
		PidFilePerm: 0644,
		LogFileName: "./log/gcsInfoCatch.log",
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[gcsInfoCatch]"},
	}
	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatal("cntxt.Search error:", err.Error())
		}
		daemon.SendCommands(d)
		return
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("cntxt.Search error:", err.Error())
	}
	if d != nil {
		return
	}
	defer cntxt.Release()
	log.Println("- - - - - - -[GCS-Info-Catch] started - - - - - - -")
	defer func() {
		log.Println("- - - - - - -[GCS-Info-Catch] exited - - - - - - -")
	}()
	//Daemon system ready

	docker_test()
	//run_handler()
}
