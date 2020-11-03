package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const startCmdDelay = time.Second

var startedRigctldCmd *exec.Cmd
var rigctldCmdStartTimer *time.Timer
var startedCmd *exec.Cmd
var cmdStartTimer *time.Timer
var serialPortStartedCmd *exec.Cmd
var serialPortCmdStartTimer *time.Timer

func doStartRigctldCmd() {
	startedRigctldCmd = exec.Command("rigctld", "-m", fmt.Sprint(rigctldModel), "-r",
		fmt.Sprint(":", serialTCPPort))
	err := startedRigctldCmd.Start()
	if err == nil {
		log.Print("rigctld started: ", startedRigctldCmd)
	} else {
		log.Error("error starting rigctld: ", err)
		startedCmd = nil
	}
	rigctldCmdStartTimer = nil
}

func startRigctldCmdIfNeeded() {
	if startedRigctldCmd != nil || disableRigctld {
		return
	}

	if rigctldCmdStartTimer != nil {
		rigctldCmdStartTimer.Stop()
	}
	rigctldCmdStartTimer = time.AfterFunc(startCmdDelay, doStartRigctldCmd)
}

func stopRigctldCmd() {
	if startedRigctldCmd == nil {
		return
	}

	if err := startedRigctldCmd.Process.Kill(); err != nil {
		log.Error("failed to stop rigctld: ", err)
	}
	_ = startedRigctldCmd.Wait()
	startedRigctldCmd = nil
}

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
	cmdStartTimer = nil
}

func startCmdIfNeeded() {
	if startedCmd != nil || runCmd == "" {
		return
	}

	if cmdStartTimer != nil {
		cmdStartTimer.Stop()
	}
	cmdStartTimer = time.AfterFunc(startCmdDelay, doStartCmd)
}

func stopCmd() {
	if startedCmd == nil {
		return
	}

	if err := startedCmd.Process.Kill(); err != nil {
		log.Error("failed to stop cmd ", runCmd, ": ", err)
	}
	startedCmd = nil
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
	serialPortCmdStartTimer = nil
}

func startSerialPortCmdIfNeeded() {
	if !enableSerialDevice || serialPortStartedCmd != nil || runCmdOnSerialPortCreated == "-" {
		return
	}

	if serialPortCmdStartTimer != nil {
		serialPortCmdStartTimer.Stop()
	}
	serialPortCmdStartTimer = time.AfterFunc(startCmdDelay, doSerialPortStartCmd)
}

func stopSerialPortCmd() {
	if serialPortStartedCmd == nil {
		return
	}

	if err := serialPortStartedCmd.Process.Kill(); err != nil {
		log.Error("failed to stop cmd ", runCmdOnSerialPortCreated, ": ", err)
	}
	serialPortStartedCmd = nil
}
