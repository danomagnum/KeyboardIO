package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	asyncKeyState  = user32.NewProc("GetAsyncKeyState")
	physicalCursor = user32.NewProc("GetPhysicalCursorPos")
)

// The array index is the PLC bit number;
var virtualKeys = [...]uintptr{
	0x26, 0x28, 0x25, 0x27, // Up, Down, Left, Right
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
	'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
	'u', 'v', 'w', 'x', 'y', 'z',
	0x20, 0x0D, // Space, Enter, Escape
}

func keyMask(read func(uintptr) uintptr) int32 {
	var mask int32
	for bit, key := range virtualKeys {
		if read(key)&0x8000 != 0 {
			mask |= 1 << bit
		}
	}
	return mask
}

type windowsDesktop struct{}

func openDesktop() (windowsDesktop, error) { return windowsDesktop{}, nil }

func (windowsDesktop) Close() {}

func (windowsDesktop) sample() (inputAssembly, error) {
	var point struct{ X, Y int32 }
	ok, _, err := physicalCursor.Call(uintptr(unsafe.Pointer(&point)))
	if ok == 0 {
		return inputAssembly{}, fmt.Errorf("desktop input unavailable; run from an interactive Windows desktop: GetPhysicalCursorPos: %w", err)
	}
	keys := keyMask(func(key uintptr) uintptr {
		state, _, _ := asyncKeyState.Call(key)
		return state
	})
	return inputAssembly{Keys: keys, X: point.X, Y: point.Y}, nil
}
