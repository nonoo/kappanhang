package main

func handleHotkey(k byte) {
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
	case '\'':
		if civControl != nil {
			if err := civControl.incRFGain(); err != nil {
				log.Error("can't increase rf gain: ", err)
			}
		}
	case ';':
		if civControl != nil {
			if err := civControl.decRFGain(); err != nil {
				log.Error("can't decrease rf gain: ", err)
			}
		}
	case ']':
		if civControl != nil {
			if err := civControl.incFreq(); err != nil {
				log.Error("can't increase freq: ", err)
			}
		}
	case '[':
		if civControl != nil {
			if err := civControl.decFreq(); err != nil {
				log.Error("can't decrease freq: ", err)
			}
		}
	case '}':
		if civControl != nil {
			if err := civControl.incTS(); err != nil {
				log.Error("can't increase ts: ", err)
			}
		}
	case '{':
		if civControl != nil {
			if err := civControl.decTS(); err != nil {
				log.Error("can't decrease ts: ", err)
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
	case 'p':
		if civControl != nil {
			if err := civControl.togglePreamp(); err != nil {
				log.Error("can't change preamp: ", err)
			}
		}
	case 'q':
		quitChan <- true
	}
}
