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

// The array index is the PLC bit number; keep existing positions stable.
var virtualKeys = [...]uintptr{
	0x26, 0x28, 0x25, 0x27, // Up, Down, Left, Right
	'W', 'A', 'S', 'D',
	0x20, 0x0D, 0x1B, // Space, Enter, Escape
	0x10, 0x11, 0x12, // Shift, Ctrl, Alt
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
