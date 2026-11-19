//go:build linux && !android

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Indices match the Windows PLC bits; aliases such as left/right Shift share a bit.
var x11Keysyms = [...][]xproto.Keysym{
	{0xff52}, {0xff54}, {0xff51}, {0xff53}, // Up, Down, Left, Right
	{'a', 'A'}, {'b', 'B'}, {'c', 'C'}, {'d', 'D'}, {'e', 'E'}, {'f', 'F'}, {'g', 'G'}, {'h', 'H'}, {'i', 'I'}, {'j', 'J'},
	{'k', 'K'}, {'l', 'L'}, {'m', 'M'}, {'n', 'N'}, {'o', 'O'}, {'p', 'P'}, {'q', 'Q'}, {'r', 'R'}, {'s', 'S'}, {'t', 'T'},
	{'u', 'U'}, {'v', 'V'}, {'w', 'W'}, {'x', 'X'}, {'y', 'Y'}, {'z', 'Z'},
	{0x20}, {0xff0d, 0xff8d}, // Space, Return/keypad Enter, Escape
}

type x11Desktop struct {
	conn        *xgb.Conn
	root        xproto.Window
	first, last xproto.Keycode
	keyBits     [256]int32
}

func openDesktop() (*x11Desktop, error) {
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		return nil, fmt.Errorf("desktop input unavailable: log into an X11 session (Wayland/XWayland is not supported)")
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("open X11 display (run in your X11 session with DISPLAY and Xauthority access): %w", err)
	}
	setup := xproto.Setup(conn)
	if conn.DefaultScreen < 0 || conn.DefaultScreen >= len(setup.Roots) {
		conn.Close()
		return nil, fmt.Errorf("X11 display has no screen %d", conn.DefaultScreen)
	}
	desktop := &x11Desktop{
		conn:  conn,
		root:  setup.DefaultScreen(conn).Root,
		first: setup.MinKeycode,
		last:  setup.MaxKeycode,
	}
	if err := desktop.refreshMapping(); err != nil {
		desktop.Close()
		return nil, err
	}
	return desktop, nil
}

func (d *x11Desktop) Close() { d.conn.Close() }

func (d *x11Desktop) refreshMapping() error {
	count := int(d.last) - int(d.first) + 1
	if count < 1 || count > 255 {
		return fmt.Errorf("invalid X11 keycode range %d-%d", d.first, d.last)
	}
	reply, err := xproto.GetKeyboardMapping(d.conn, d.first, byte(count)).Reply()
	if err != nil {
		return fmt.Errorf("read X11 keyboard mapping: %w", err)
	}
	if reply == nil {
		return fmt.Errorf("X11 keyboard mapping reply missing")
	}
	d.keyBits, err = mapX11Keys(d.first, count, reply)
	return err
}

func mapX11Keys(first xproto.Keycode, count int, reply *xproto.GetKeyboardMappingReply) ([256]int32, error) {
	var bits [256]int32
	width := int(reply.KeysymsPerKeycode)
	if count < 1 || count > len(bits)-int(first) || width == 0 || len(reply.Keysyms) != count*width {
		return bits, fmt.Errorf("invalid X11 keyboard mapping")
	}
	for i, sym := range reply.Keysyms {
		for bit, aliases := range x11Keysyms {
			for _, alias := range aliases {
				if sym == alias {
					bits[int(first)+i/width] |= 1 << bit
				}
			}
		}
	}
	return bits, nil
}

func x11KeyMask(bits [256]int32, keys []byte) int32 {
	var mask int32
	for code, bit := range bits {
		if bit != 0 && keys[code/8]&(1<<(code%8)) != 0 {
			mask |= bit
		}
	}
	return mask
}

func (d *x11Desktop) sample() (inputAssembly, error) {
	refresh := false
	for {
		event, err := d.conn.PollForEvent()
		if err != nil {
			return inputAssembly{}, fmt.Errorf("read X11 event: %w", err)
		}
		if event == nil {
			break
		}
		if change, ok := event.(xproto.MappingNotifyEvent); ok && change.Request == xproto.MappingKeyboard {
			refresh = true
		}
	}
	if refresh {
		if err := d.refreshMapping(); err != nil {
			return inputAssembly{}, err
		}
	}
	keyReq := xproto.QueryKeymap(d.conn)
	pointerReq := xproto.QueryPointer(d.conn, d.root)
	keys, err := keyReq.Reply()
	if err != nil {
		return inputAssembly{}, fmt.Errorf("read X11 key state: %w", err)
	}
	if keys == nil || len(keys.Keys) != 32 {
		return inputAssembly{}, fmt.Errorf("invalid X11 key state reply")
	}
	pointer, err := pointerReq.Reply()
	if err != nil {
		return inputAssembly{}, fmt.Errorf("read X11 cursor position: %w", err)
	}
	if pointer == nil {
		return inputAssembly{}, fmt.Errorf("X11 cursor position reply missing")
	}
	return inputAssembly{Keys: x11KeyMask(d.keyBits, keys.Keys), X: int32(pointer.RootX), Y: int32(pointer.RootY)}, nil
}
