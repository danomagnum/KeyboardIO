package main

import "testing"

func TestKeyMaskUsesCurrentDownState(t *testing.T) {
	got := keyMask(func(key uintptr) uintptr {
		switch key {
		case 0x26:
			return 0x8000 // Up held
		case 'A':
			return 0x8001 // A held and previously pressed
		case 0x12:
			return 0xffff8000 // Alt held, with upper bits set
		case 0x20:
			return 1 // Space was pressed but is no longer held
		default:
			return 0
		}
	})
	if want := int32(1<<0 | 1<<5 | 1<<13); got != want {
		t.Fatalf("key mask = %#x, want %#x", got, want)
	}
}
