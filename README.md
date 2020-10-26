# kappanhang

kappanhang remotely opens audio channels and a serial port to an Icom RS-BA1
server. The app is mainly developed for connecting to the Icom IC-705
transceiver, which has built-in Wi-Fi and RS-BA1 server. All features of the
protocol are implemented including packet retransmission on packet loss.

kappanhang currently only supports Linux, but support for other platforms can
be easily added if anyone is interested (volunteers needed).

## Compiling

You'll need Go installed on your computer.

```
go get https://github.com/nonoo/kappanhang
go install https://github.com/nonoo/kappanhang
```

## Required settings on the RS-BA1 server (the transceiver)

- Set the username to **beer** and the password to **beerbeer**.
  These are fixed as the password encoding of the RS-BA1 protocol has not been
  decrypted yet. See [passcode.txt](passcode.txt) for more information.
- Leave the **UDP ports** on their default values:
  - Control port: 50001
  - Serial port: 50002
  - Audio port: 50003
- Leave the **Internet access line** on the default **FTTH** value.

## Running

You can get the available command line parameters with the **-h** command line
argument.

If no command line arguments are set, then the app will try to connect to the
host **ic-705** (ic-705.local or ic-705.localdomain).

After it is connected and logged in:

- TODO: audio
- It starts a **TCP server** on port 4533 for exposing the **serial port**.
  This can be used for controlling the server (the transceiver) with
  [Hamlib](https://hamlib.github.io/):

  ```
  rigctld -m 3085 -r 127.0.0.1:4533
  ```

  3085 is the model number for the Icom IC-705. `rigctld` will connect to
  kappanhang's TCP serial port server, and waits connections on it's default
  TCP port 4532.

  To use this with for example [WSJT-X](https://physics.princeton.edu/pulsar/K1JT/wsjtx.html),
  open WSJT-X settings, go to the *Radio* tab, set the *rig type* to **Hamlib NET
  rigctl**, and the *Network server* to `127.0.0.1:4532`.

If the `-s` command line argument is specified, then kappanhang will create a
virtual serial port, so other apps which don't support Hamlib can access the
transceiver directly. Look at the app log to find out the name of the virtual
serial port.

## Donations

If you find this app useful then [buy me a beer](https://paypal.me/ha2non). :)
