package main

import (
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const startCmdDelay = time.Second

type cmdRunner struct {
	restartNeeded  chan bool
	runEndNeeded   chan bool
	runEndFinished chan bool
}

var runCmdRunner cmdRunner
var serialCmdRunner cmdRunner

func (c *cmdRunner) kill(cmd *exec.Cmd) {
	err := cmd.Process.Kill()
	if err != nil {
		_ = cmd.Process.Signal(syscall.SIGKILL)
	}
}

func (c *cmdRunner) run(cmdLine string) {
	var cmd *exec.Cmd

	defer func() {
		if cmd != nil {
			c.kill(cmd)
		}
		c.runEndFinished <- true
	}()

	s := strings.Split(cmdLine, " ")

	for {
		select {
		case <-c.runEndNeeded:
			return
		case <-time.After(startCmdDelay):
		}

		cmd = exec.Command(s[0], s[1:]...)
		err := cmd.Start()
		if err != nil {
			log.Error("error starting ", cmd)
			continue
		}

		log.Print("started: ", cmd)

		finishedChan := make(chan error)
		go func() {
			finishedChan <- cmd.Wait()
		}()

		select {
		case <-c.restartNeeded:
			log.Debug("restarting ", cmd)
			c.kill(cmd)
		case err := <-finishedChan:
			if err != nil {
				log.Error(cmd, " error: ", err)
			}
		case <-c.runEndNeeded:
			return
		}
	}
}

func (c *cmdRunner) startIfNeeded(cmdLine string) {
	if c.runEndNeeded != nil || cmdLine == "" || cmdLine == "-" {
		return
	}

	c.restartNeeded = make(chan bool)
	c.runEndNeeded = make(chan bool)
	c.runEndFinished = make(chan bool)
	go c.run(cmdLine)
}

// func (c *cmdRunner) restart() {
// 	if c.restartNeeded == nil {
// 		return
// 	}

// 	c.restartNeeded <- true
// }

func (c *cmdRunner) stop() {
	if c.runEndNeeded == nil {
		return
	}

	c.runEndNeeded <- true
	<-c.runEndFinished
}
