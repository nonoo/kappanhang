# kappanhang

kappanhang remotely opens audio channels and a serial port to an Icom RS-BA1
server. The app is mainly developed for connecting to the Icom IC-705
transceiver, which has built-in Wi-Fi and RS-BA1 server. All features of the
protocol are implemented including packet retransmission on packet loss.

<p align="center"><img src="demo.gif?raw=true"/></p>

kappanhang currently only supports Linux, but support for other platforms can
be easily added if anyone is interested (volunteers needed).

## Compiling

You'll need Go installed on your computer.

```
go get https://github.com/nonoo/kappanhang
go install https://github.com/nonoo/kappanhang
```

## Required settings on the RS-BA1 server (the transceiver)

You can find these settings on the Icom IC-705 in: `Menu -> Set -> WLAN set ->
Remote settings`.

- Make sure **Network control** is turned on.
- Set the **Network user 1** username to `beer` and the password to
  `beerbeer`. These are fixed as the password encoding of the RS-BA1
  protocol has not been decrypted yet. See [passcode.txt](passcode.txt) for
  more information.
- Leave the **UDP ports** on their default values:
  - Control port: `50001`
  - Serial port: `50002`
  - Audio port: `50003`
- Leave the **Internet access line** on the default `FTTH` value.

Make sure the **DATA MOD** (you can find this setting on the Icom IC-705 in:
`Menu -> Set -> Connectors -> MOD Input -> DATA MOD`) is set to `WLAN`.

## Running

You can get the available command line parameters with the `-h` command line
argument.

If no command line arguments are set, then the app will try to connect to the
host **ic-705** (ic-705.local or ic-705.localdomain).

After it is connected and logged in:

- Creates a virtual PulseAudio **sound card** (48kHz, s16le, mono). This can be
  used to record/play audio from/to the server (the radio). You can also set
  this sound card in [WSJT-X](https://physics.princeton.edu/pulsar/K1JT/wsjtx.html).

  If you want to listen to the audio coming from this sound card in real time,
  then you can create a [PulseAudio loopback](https://github.com/alentoghostflame/Python-Pulseaudio-Loopback-Tool)
  between the kappanhang sound card and your real sound card. You can also
  create a loopback for your microphone using this tool, so you'll be able to
  transmit your voice.
- Starts a **TCP server** on port `4533` for exposing the **serial port**.
  This can be used for controlling the server (the transceiver) with
  [Hamlib](https://hamlib.github.io/):

  ```
  rigctld -m 3085 -r 127.0.0.1:4533
  ```

  3085 is the model number for the Icom IC-705. `rigctld` will connect to
  kappanhang's TCP serial port server, and waits connections on it's default
  TCP port `4532`.

  To use this with for example [WSJT-X](https://physics.princeton.edu/pulsar/K1JT/wsjtx.html),
  open WSJT-X settings, go to the *Radio* tab, set the *rig type* to `Hamlib NET
  rigctl`, and the *Network server* to `127.0.0.1:4532`.

If the `-s` command line argument is specified, then kappanhang will create a
**virtual serial port**, so other apps which don't support Hamlib can access
the transceiver directly. Look at the app log to find out the name of the
virtual serial port.

### Status bar

kappanhang displays a "realtime" status bar (when the audio/serial connection
is up) with the following info:

- First status bar line:
  - `state`: RX/TX/TUNE depending on the PTT status
  - `freq`: operating frequency in MHz, mode (LSB/USB/FM...), active filter

- Second status bar line:
  - `up`: how long the audio/serial connection is active
  - `rtt`: roundtrip communication latency with the server
  - `up/down`: currently used upload/download bandwidth (only considering UDP
    payload to/from the server)
  - `retx`: audio/serial retransmit request count to/from the server
  - `lost`: lost audio/serial packet count from the server

Data for the first status bar line is acquired by monitoring CiV traffic in
the serial stream.

`retx` and `lost` are displayed in a 1 minute window, which means they will be
reset to 0 if they don't increase for 1 minute. A `retx` value other than 0
indicates issues with the connection (probably a poor Wi-Fi connection), but
if `loss` stays 0 then the issues were fixed using packet retransmission.
`loss` indicates failed retransmit sequences, so packet loss. This can cause
audio and serial communication disruptions.

If status bar interval (can be changed with the `-i` command line
argument) is equal to or above 1 second, then the realtime status bar will be
disabled and the contents of the second line of the status bar will be written
as new console log lines. This is also the case if a Unix/VT100 terminal is
not available.

## Authors

- Norbert Varga HA2NON [nonoo@nonoo.hu](mailto:nonoo@nonoo.hu)
- Akos Marton ES1AKOS

## Donations

If you find this app useful then [buy me a beer](https://paypal.me/ha2non). :)
