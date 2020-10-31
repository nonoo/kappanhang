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
	case 't':
		if civControl != nil {
			if err := civControl.toggleTune(); err != nil {
				log.Error("can't toggle tune: ", err)
			}
		}
	case '+':
		if civControl != nil {
			if err := civControl.incPwr(); err != nil {
				log.Error("can't increase power: ", err)
			}
		}
	case '-':
		if civControl != nil {
			if err := civControl.decPwr(); err != nil {
				log.Error("can't decrease power: ", err)
			}
		}
	case '>':
		if civControl != nil {
			if err := civControl.incFreq(0.000001); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case '<':
		if civControl != nil {
			if err := civControl.decFreq(0.000001); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case '.':
		if civControl != nil {
			if err := civControl.incFreq(0.00001); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case ',':
		if civControl != nil {
			if err := civControl.decFreq(0.00001); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case '"':
		if civControl != nil {
			if err := civControl.incFreq(0.0001); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case ':':
		if civControl != nil {
			if err := civControl.decFreq(0.0001); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case '\'':
		if civControl != nil {
			if err := civControl.incFreq(0.001); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case ';':
		if civControl != nil {
			if err := civControl.decFreq(0.001); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case '}':
		if civControl != nil {
			if err := civControl.incFreq(0.01); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case '{':
		if civControl != nil {
			if err := civControl.decFreq(0.01); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case ']':
		if civControl != nil {
			if err := civControl.incFreq(0.1); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case '[':
		if civControl != nil {
			if err := civControl.decFreq(0.1); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case 'm':
		if civControl != nil {
			if err := civControl.incOperatingMode(); err != nil {
				log.Error("can't change mode: ", err)
			}
		}
	case 'n':
		if civControl != nil {
			if err := civControl.decOperatingMode(); err != nil {
				log.Error("can't change mode: ", err)
			}
		}
	case 'f':
		if civControl != nil {
			if err := civControl.incFilter(); err != nil {
				log.Error("can't change filter: ", err)
			}
		}
	case 'd':
		if civControl != nil {
			if err := civControl.decFilter(); err != nil {
				log.Error("can't change filter: ", err)
			}
		}
	case 'D':
		if civControl != nil {
			if err := civControl.toggleDataMode(); err != nil {
				log.Error("can't change datamode: ", err)
			}
		}
	case 'b':
		if civControl != nil {
			if err := civControl.incBand(); err != nil {
				log.Error("can't change band: ", err)
			}
		}
	case 'v':
		if civControl != nil {
			if err := civControl.decBand(); err != nil {
				log.Error("can't change band: ", err)
			}
		}
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
