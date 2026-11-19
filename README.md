# KeyboardIO

KeyboardIO publishes desktop keyboard states and mouse position to a PLC using
gologix EtherNet/IP Class 1 cyclic I/O. It runs on Windows and Linux (X11).

## Run

From an interactive Windows desktop:

```powershell
go build .
.\keyboardio.exe -bind 192.168.1.50
```

From a terminal in your Linux X11 desktop:

```sh
go build .
./keyboardio -bind 192.168.1.50
```

Linux uses the current X11 display (`DISPLAY`) and its normal Xauthority
credentials. No root privileges or cgo are required. Log into an X11 session
if `echo "$XDG_SESSION_TYPE"` reports `wayland`; Wayland and XWayland are not
supported for desktop-wide input.

Replace `192.168.1.50` with the PC's IPv4 address on the PLC network. Without
`-bind`, the server listens only on `127.0.0.1`. If needed, allow inbound
TCP 44818 and UDP 2222 from the PLC in your host firewall. The PLC also receives
cyclic I/O on UDP 2222. Use a trusted controls network: EtherNet/IP on this
server does not authenticate or encrypt the input data.

The program polls every 5 ms by default (`-poll 5ms`), including while another
app is focused. It publishes only the keys listed below, does not record inputs, and
does not inject keyboard or mouse events. Press Ctrl+C to stop it. Run it in
the signed-in user's session, not as a service.

## Studio 5000

Add a **Generic Ethernet Module** using the PC's IP address. Start with a
**20 ms RPI** and use a **unicast** connection.

| Setting | Value |
| --- | --- |
| Comm Format | Data - DINT |
| Input instance / size | 101 / 3 DINTs |
| Output instance / size | 100 / 1 DINT (ignored) |
| Configuration instance / size | 102 / 0 bytes |

These are suggested assembly IDs for the scanner; gologix routes by the path
prefix rather than by these IDs. For a bridge/backplane setup, the endpoint
is also registered at slot 0.

| PLC input | Value |
| --- | --- |
| `KeyboardIO:I.Data[0]` | Key state bitmask |
| `KeyboardIO:I.Data[1]` | Mouse X, signed physical desktop pixels |
| `KeyboardIO:I.Data[2]` | Mouse Y, signed physical desktop pixels |

On Windows, coordinates are relative to the primary display origin and can be
negative on other monitors. On X11, they are relative to the root window of the
screen containing the pointer, typically the upper-left corner of the combined
desktop. Bits are 1 while the key is held and 0 when released. The wire payload
is 12 bytes, little-endian, on both platforms.

| Bit | Key | Bit | Key |
| --- | --- | --- | --- |
| 0 | Up | 7 | D |
| 1 | Down | 8 | Space |
| 2 | Left | 9 | Enter |
| 3 | Right | 10 | Escape |
| 4 | W | 11 | Shift |
| 5 | A | 12 | Ctrl |
| 6 | S | 13 | Alt |

Bits 14-31 are reserved and always zero. For example, `KeyboardIO:I.Data[0].0`
is the Up arrow. The module sends snapshots, not latched events, so a quick tap
can be missed between samples or PLC updates. Monitor the connection status in
PLC logic; do not treat the last received state as fresh after a connection loss.
This is not safety-rated I/O and should not be used for safety functions.
