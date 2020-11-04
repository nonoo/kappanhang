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
	case '0':
		if civControl != nil {
			if err := civControl.setPwr(0); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '1':
		if civControl != nil {
			if err := civControl.setPwr(10); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '2':
		if civControl != nil {
			if err := civControl.setPwr(20); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '3':
		if civControl != nil {
			if err := civControl.setPwr(30); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '4':
		if civControl != nil {
			if err := civControl.setPwr(40); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '5':
		if civControl != nil {
			if err := civControl.setPwr(50); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '6':
		if civControl != nil {
			if err := civControl.setPwr(60); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '7':
		if civControl != nil {
			if err := civControl.setPwr(70); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '8':
		if civControl != nil {
			if err := civControl.setPwr(80); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case '9':
		if civControl != nil {
			if err := civControl.setPwr(90); err != nil {
				log.Error("can't set power: ", err)
			}
		}
	case ')':
		if civControl != nil {
			if err := civControl.setPwr(100); err != nil {
				log.Error("can't set power: ", err)
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
	case '"':
		if civControl != nil {
			if err := civControl.incSQL(); err != nil {
				log.Error("can't increase sql: ", err)
			}
		}
	case ':':
		if civControl != nil {
			if err := civControl.decSQL(); err != nil {
				log.Error("can't decrease sql: ", err)
			}
		}
	case '.':
		if civControl != nil {
			if err := civControl.incNR(); err != nil {
				log.Error("can't increase nr: ", err)
			}
		}
	case ',':
		if civControl != nil {
			if err := civControl.decNR(); err != nil {
				log.Error("can't decrease nr: ", err)
			}
		}
	case '/':
		if civControl != nil {
			if err := civControl.toggleNR(); err != nil {
				log.Error("can't toggle nr: ", err)
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
