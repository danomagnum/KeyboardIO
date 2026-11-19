//go:build linux && !android

package main

import (
	"testing"

	"github.com/jezek/xgb/xproto"
)

func TestX11KeyState(t *testing.T) {
	mapping := xproto.GetKeyboardMappingReply{
		KeysymsPerKeycode: 2,
		Keysyms: []xproto.Keysym{
			0xff52, 0, // 13: Up
			'z', 'Z', // 14: unmapped, but held
			0, 'A', // 15: A found in the shifted column
			0xffe1, 0, // 16: Left Shift
			0xffe2, 0, // 17: Right Shift
			0xff1b, 0, // 18: Escape, not held
		},
	}
	bits, err := mapX11Keys(13, 6, &mapping)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]byte, 32)
	keys[1] = 0xe0 // keycodes 13, 14, 15
	for _, shift := range []byte{1, 2, 3} {
		keys[2] = shift
		if got, want := x11KeyMask(bits, keys), int32(1<<0|1<<5|1<<11); got != want {
			t.Errorf("key mask = %#x, want %#x", got, want)
		}
	}
	clear(keys)
	if got := x11KeyMask(bits, keys); got != 0 {
		t.Errorf("released key mask = %#x, want 0", got)
	}
}

func TestX11RejectsTruncatedMapping(t *testing.T) {
	mapping := xproto.GetKeyboardMappingReply{KeysymsPerKeycode: 2, Keysyms: []xproto.Keysym{'a'}}
	if _, err := mapX11Keys(8, 1, &mapping); err == nil {
		t.Fatal("accepted a truncated keyboard mapping")
	}
}
