# kappanhang

kappanhang remotely opens audio channels and a serial port to an Icom RS-BA1
server. The app is mainly developed for connecting to the Icom IC-705
transceiver, which has built-in Wi-Fi and RS-BA1 server. All features of the
protocol are implemented including packet retransmission on packet loss.

<p align="center"><img src="demo.gif?raw=true"/></p>

kappanhang currently only supports Linux, but support for other platforms can
be easily added if anyone is interested and has the skills (volunteers
needed, as I'm only developing the Linux version).

You can find pre-built versions for Linux/x64 on the GitHub project page of
kappanhang.

## Compiling

You'll need Go installed on your computer.

```
go get https://github.com/nonoo/kappanhang
go install https://github.com/nonoo/kappanhang
```

## Required settings on the RS-BA1 server (the transceiver)

- Make sure network settings (on the Icom IC-705 in: `Menu -> Set ->
  WLAN set -> Remote settings`) are the following:
  - **Network control** is turned on.
  - **UDP ports** are on their default values:
    - Control port: `50001`
    - Serial port: `50002`
    - Audio port: `50003`
  - **Internet access line** is on the default `FTTH` value.
- Make sure the following settings are set:
  - **DATA MOD** is set to `WLAN` (on the Icom IC-705 in: `Menu -> Set ->
    Connectors -> MOD Input -> DATA MOD`)
  - **CI-V Address** is on the default `A4h` value (on the Icom IC-705 in:
    `Menu -> Set -> Connectors -> CI-V`.

## Running

You can get the available command line parameters with the `-h` command line
argument.

If no command line arguments are set, then the app will try to connect to the
host **ic-705** (ic-705.local or ic-705.localdomain) with the username `beer`
and password `beerbeer`. You can set the username with the `-u` and the
password with the `-p` command line arguments.

After it is connected and logged in:

- Creates a virtual PulseAudio **sound card** (48kHz, s16le, mono). This can be
  used to record/play audio from/to the server (the transceiver). You can also
  set this sound card in [WSJT-X](https://physics.princeton.edu/pulsar/K1JT/wsjtx.html).
- Starts an **internal rigctld** server. This can be used for controlling the
  server (the transceiver) with [Hamlib](https://hamlib.github.io/) (`rigctl`)
  clients. This internal rigctld is needed for more reliable rigctl
  communication, as the original rigctld is very sensitive to timeouts.

  To use this with for example [WSJT-X](https://physics.princeton.edu/pulsar/K1JT/wsjtx.html),
  open WSJT-X settings, go to the *Radio* tab, set the *rig type* to `Hamlib
  NET rigctl`, and the *Network server* to `localhost`.
- Starts a **TCP server** on port `4533` for exposing the **serial port**.
  This can be used for an externally launched `rigctld` for example.

### Virtual serial port

If the `-s` command line argument is specified, then kappanhang will create a
**virtual serial port**, so other apps which don't support Hamlib can access
the transceiver directly. Look at the app log to find out the name of the
virtual serial port. It will be something like `/tmp/kappanhang-IC-705.pty`
(the server's name appended to the string *kappanhang*). After the virtual
serial port is created, the command specified with `-o` will be ran, which is
`socat /tmp/kappanhang-IC-705.pty /tmp/vmware.pty` by default. Running the
command can be disabled with `-o -`. The command is only executed once, as the
virtual serial port will stay opened even if the RS-BA1 server disconnects.
I use this command to link a COM port in a Windows OS running in VMware to
the virtual serial port, so I can use the original RS-BA1 software remote
control GUI.

### Status bar

kappanhang displays a "realtime" status bar (when the audio/serial connection
is up) with the following info:

- First status bar line:
  - `MON/REC`: current status of the audio monitor (see the *Hotkeys* section
    in this README for more information about this feature)
  - `filter`: active filter (FIL1, FIL2 etc.)
  - `preamp`: PAMP0 means the preamp is off
  - `AGC`: AGC state (F - fast, M - middle, S - slow)
  - `rfg`: RF gain in percent
  - `sql`: squelch level in percent
  - `nr`: noise reduction level in percent

- Second status bar line:
  - `S meter`: periodically refreshed S meter value, OVF is displayed on
    overflow, displays TX on transmit (or TUNE)
  - `freq`: operating frequency in MHz
  - `TS`: tuning step
  - `mode`: LSB/USB/FM etc. *-D* indicates data mode
  - `SPLIT/DUP-/DUP+`: displayed when split/DUP operation is active, the TX
    frequency is also displayed in split mode
  - `voltage`: drain voltage of the final amplifier MOS-FETs, updated when a
    TX/TUNE is over
  - `txpwr`: current transmit power setting in percent
  - `swr`: reported SWR (only displayed during TX)

- Third status bar line:
  - `up`: how long the audio/serial connection is active
  - `rtt`: roundtrip communication latency with the server
  - `up/down`: currently used upload/download bandwidth (only considering UDP
    payload to/from the server)
  - `retx`: audio/serial retransmit request count to/from the server
  - `lost`: lost audio/serial packet count from the server

Data for the first 2 status bar lines are acquired by monitoring CiV traffic
in the serial stream. S value and OVF are queried periodically, but these
queries/replies are filtered from the serial data stream sent to the TCP
serial port server and to the virtual serial port.

`retx` and `lost` are displayed in a 1 minute window, which means they will be
reset to 0 if they don't increase for 1 minute. A `retx` value other than 0
indicates issues with the connection (probably a poor Wi-Fi connection), but
if `loss` stays 0 then the issues were fixed using packet retransmission.
`loss` indicates failed retransmit sequences, so packet loss. This can cause
audio and serial communication disruptions.

If status bar interval (can be changed with the `-i` command line
argument) is equal to or above 1 second, then the realtime status bar will be
disabled and the contents of the last line of the status bar will be written
as new console log lines. This is also the case if a Unix/VT100 terminal is
not available.

### Hotkeys

- `q` (quit): closes the app
- `l` (listen): toggles audio stream playback to the default sound device.
  This is useful for quickly listening into the audio stream coming from the
  server (the transceiver).
- `space`: toggles PTT and audio stream recording from the default sound
  device. You can transmit your own voice using a mic attached to your
  computer for example.

Some basic CAT control hotkeys are also supported:

- `t`: toggles the tune process
- `+`: increases TX power
- `-`: decreases TX power
- `0` to `9`: set TX power in 10% steps
- `)`: set TX power to 100%
- `[`, `]`: decreases, increases frequency
- `{`, `}`: decreases, increases tuning step
- `;`, `'`: decreases, increases RF gain
- `!` to `(` (shift + numbers): set RF gain in 10% steps
- `:`, `"`: decreases, increases squelch level
- `,`, `.`: decreases, increases noise reduction level
- `/`: toggles noise reduction
- `n`, `m`: cycles through operating modes
- `d`, `f`: cycles through filters
- `D`: toggles data mode
- `v`, `b`: cycles through bands
- `p`: toggles preamp
- `a`: toggles AGC
- `o`: toggles VFO A/B
- `s`: toggles split/DUP+- operation

## Icom IC-705 Wi-Fi notes

Note that the built-in Wi-Fi in the Icom IC-705 has **very limited range**,
and **sensitive to interference**. If you see a lot of retransmits in the log,
or packet loss, then:

- Place the IC-705 close to your Wi-Fi AP/router, or use a Wi-Fi range
  extender device
- Make sure the Wi-Fi bandwith is set to max. 20Mhz in the Wi-Fi router (see
explanation [here](https://superuser.com/questions/542191/does-moving-my-router-from-20mhz-to-40-mhz-increase-my-wireless-speed))
- Try switching Wi-Fi channel on your Wi-Fi router. Only channels 1, 6 or 11
should be used (see explanation [here](https://www.metageek.com/training/resources/why-channels-1-6-11.html))

Sometimes rebooting the transceiver helps, as the network stack in the IC-705
is not quite free of bugs. :)

## Contributors

- Norbert Varga HA2NON [nonoo@nonoo.hu](mailto:nonoo@nonoo.hu)
- Akos Marton ES1AKOS
- W6EL (passcode algorithm)

## Donations

If you find this app useful then [buy me a beer](https://paypal.me/ha2non). :)
