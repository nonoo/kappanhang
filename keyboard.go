package main

import (
	"os"
	"os/exec"
)

type keyboardStruct struct {
}

var keyboard keyboardStruct

func (s *keyboardStruct) loop() {
	var b []byte = make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if n > 0 && err == nil {
			if b[0] == 'l' {
				if audio.togglePlaybackToDefaultSoundcardChan != nil {
					// Non-blocking send to channel.
					select {
					case audio.togglePlaybackToDefaultSoundcardChan <- true:
					default:
					}
				}
			}
		}
	}
}

func (s *keyboardStruct) init() {
	if err := exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run(); err != nil {
		log.Error("can't disable input buffering")
	}
	if err := exec.Command("stty", "-F", "/dev/tty", "-echo").Run(); err != nil {
		log.Error("can't disable displaying entered characters")
	}

	go s.loop()
}

func (s *keyboardStruct) deinit() {
	_ = exec.Command("stty", "-F", "/dev/tty", "echo").Run()
}
