package main

import (
	"os/exec"
	"strings"
	"time"
)

const startCmdDelay = time.Second

var startedCmd *exec.Cmd

func doStartCmd() {
	c := strings.Split(runCmd, " ")
	startedCmd = exec.Command(c[0], c[1:]...)
	err := startedCmd.Start()
	if err == nil {
		log.Print("cmd started: ", runCmd)
	} else {
		log.Error("error starting ", runCmd, " - ", err)
		startedCmd = nil
	}
}

func startCmdIfNeeded() {
	if startedCmd != nil || runCmd == "-" {
		return
	}

	time.AfterFunc(startCmdDelay, doStartCmd)
}

func stopCmd() {
	if startedCmd == nil {
		return
	}
	if err := startedCmd.Process.Kill(); err != nil {
		log.Error("failed to stop cmd ", runCmd)
	}
}
