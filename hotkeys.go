package main

import "fmt"

func handleHotkey(k byte) {
	switch k {
	case 'l':
		audio.togglePlaybackToDefaultSoundcard()
	case ' ':
		audio.toggleRecFromDefaultSoundcard()
	case 't':
		if err := civControl.toggleTune(); err != nil {
			log.Error("can't toggle tune: ", err)
		}
	case '+':
		if err := civControl.incPwr(); err != nil {
			log.Error("can't increase power: ", err)
		}
	case '-':
		if err := civControl.decPwr(); err != nil {
			log.Error("can't decrease power: ", err)
		}
	case '0':
		if err := civControl.setPwr(0); err != nil {
			log.Error("can't set power: ", err)
		}
	case '1':
		if err := civControl.setPwr(10); err != nil {
			log.Error("can't set power: ", err)
		}
	case '2':
		if err := civControl.setPwr(20); err != nil {
			log.Error("can't set power: ", err)
		}
	case '3':
		if err := civControl.setPwr(30); err != nil {
			log.Error("can't set power: ", err)
		}
	case '4':
		if err := civControl.setPwr(40); err != nil {
			log.Error("can't set power: ", err)
		}
	case '5':
		if err := civControl.setPwr(50); err != nil {
			log.Error("can't set power: ", err)
		}
	case '6':
		if err := civControl.setPwr(60); err != nil {
			log.Error("can't set power: ", err)
		}
	case '7':
		if err := civControl.setPwr(70); err != nil {
			log.Error("can't set power: ", err)
		}
	case '8':
		if err := civControl.setPwr(80); err != nil {
			log.Error("can't set power: ", err)
		}
	case '9':
		if err := civControl.setPwr(90); err != nil {
			log.Error("can't set power: ", err)
		}
	case ')':
		if err := civControl.setPwr(100); err != nil {
			log.Error("can't set power: ", err)
		}
	case '!':
		if err := civControl.setRFGain(10); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '@':
		if err := civControl.setRFGain(20); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '#':
		if err := civControl.setRFGain(30); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '$':
		if err := civControl.setRFGain(40); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '%':
		if err := civControl.setRFGain(50); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '^':
		if err := civControl.setRFGain(60); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '&':
		if err := civControl.setRFGain(70); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '*':
		if err := civControl.setRFGain(80); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '(':
		if err := civControl.setRFGain(90); err != nil {
			log.Error("can't set rfgain: ", err)
		}
	case '\'':
		if err := civControl.incRFGain(); err != nil {
			log.Error("can't increase rf gain: ", err)
		}
	case ';':
		if err := civControl.decRFGain(); err != nil {
			log.Error("can't decrease rf gain: ", err)
		}
	case '"':
		if err := civControl.incSQL(); err != nil {
			log.Error("can't increase sql: ", err)
		}
	case ':':
		if err := civControl.decSQL(); err != nil {
			log.Error("can't decrease sql: ", err)
		}
	case '.':
		if err := civControl.incNR(); err != nil {
			log.Error("can't increase nr: ", err)
		}
	case ',':
		if err := civControl.decNR(); err != nil {
			log.Error("can't decrease nr: ", err)
		}
	case '/':
		if err := civControl.toggleNR(); err != nil {
			log.Error("can't toggle nr: ", err)
		}
	case ']':
		if err := civControl.incFreq(); err != nil {
			log.Error("can't increase freq: ", err)
		}
	case '[':
		if err := civControl.decFreq(); err != nil {
			log.Error("can't decrease freq: ", err)
		}
	case '}':
		if err := civControl.incTS(); err != nil {
			log.Error("can't increase ts: ", err)
		}
	case '{':
		if err := civControl.decTS(); err != nil {
			log.Error("can't decrease ts: ", err)
		}
	case 'm':
		if err := civControl.incOperatingMode(); err != nil {
			log.Error("can't change mode: ", err)
		}
	case 'n':
		if err := civControl.decOperatingMode(); err != nil {
			log.Error("can't change mode: ", err)
		}
	case 'f':
		if err := civControl.incFilter(); err != nil {
			log.Error("can't change filter: ", err)
		}
	case 'd':
		if err := civControl.decFilter(); err != nil {
			log.Error("can't change filter: ", err)
		}
	case 'D':
		if err := civControl.toggleDataMode(); err != nil {
			log.Error("can't change datamode: ", err)
		}
	case 'b':
		if err := civControl.incBand(); err != nil {
			log.Error("can't change band: ", err)
		}
	case 'v':
		if err := civControl.decBand(); err != nil {
			log.Error("can't change band: ", err)
		}
	case 'p':
		if err := civControl.togglePreamp(); err != nil {
			log.Error("can't change preamp: ", err)
		}
	case 'a':
		if err := civControl.toggleAGC(); err != nil {
			log.Error("can't change agc: ", err)
		}
	case 'o':
		if err := civControl.toggleVFO(); err != nil {
			log.Error("can't change vfo: ", err)
		}
	case 's':
		if err := civControl.toggleSplit(); err != nil {
			log.Error("can't change split: ", err)
		}
	case '\n':
		if statusLog.isRealtime() {
			statusLog.mutex.Lock()
			statusLog.clearInternal()
			fmt.Println()
			statusLog.mutex.Unlock()
			statusLog.print()
		}
	case 'q':
		quitChan <- true
	}
}
