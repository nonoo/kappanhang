// +build linux

package main

import (
	"os"
	"os/exec"
)

type keyboardStruct struct {
}

var keyboard keyboardStruct

func (s *keyboardStruct) handleKey(k byte) {
	switch k {
	case 'l':
		audio.togglePlaybackToDefaultSoundcard()
	case ' ':
		audio.toggleRecFromDefaultSoundcard()
	}
}

func (s *keyboardStruct) loop() {
	var b []byte = make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if n > 0 && err == nil {
			s.handleKey(b[0])
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
