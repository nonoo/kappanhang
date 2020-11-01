package main

import (
	"os/exec"
	"strings"
	"time"
)

const startCmdDelay = time.Second

var startedCmd *exec.Cmd
var serialPortStartedCmd *exec.Cmd

func doStartCmd() {
	c := strings.Split(runCmd, " ")
	startedCmd = exec.Command(c[0], c[1:]...)
	err := startedCmd.Start()
	if err == nil {
		log.Print("cmd started: ", runCmd)
	} else {
		log.Error("error starting ", runCmd, ": ", err)
		startedCmd = nil
	}
}

func startCmdIfNeeded() {
	if startedCmd != nil || runCmd == "-" {
		return
	}

	time.AfterFunc(startCmdDelay, doStartCmd)
}

func doSerialPortStartCmd() {
	c := strings.Split(runCmdOnSerialPortCreated, " ")
	serialPortStartedCmd = exec.Command(c[0], c[1:]...)
	err := serialPortStartedCmd.Start()
	if err == nil {
		log.Print("cmd started: ", runCmdOnSerialPortCreated)
	} else {
		log.Error("error starting ", runCmdOnSerialPortCreated, ": ", err)
		serialPortStartedCmd = nil
	}
}

func startSerialPortCmdIfNeeded() {
	if serialPortStartedCmd != nil || runCmdOnSerialPortCreated == "-" {
		return
	}

	time.AfterFunc(startCmdDelay, doSerialPortStartCmd)
}

func stopCmd() {
	if startedCmd != nil {
		if err := startedCmd.Process.Kill(); err != nil {
			log.Error("failed to stop cmd ", runCmd, ": ", err)
		}
		startedCmd = nil
	}
}

func stopSerialPortCmd() {
	if serialPortStartedCmd != nil {
		if err := serialPortStartedCmd.Process.Kill(); err != nil {
			log.Error("failed to stop cmd ", runCmdOnSerialPortCreated, ": ", err)
		}
		serialPortStartedCmd = nil
	}
}
