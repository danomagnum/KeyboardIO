//go:build windows || (linux && !android)

package main

import (
	"bytes"
	"testing"

	"github.com/danomagnum/gologix"
)

func TestInputAssemblyWireFormat(t *testing.T) {
	endpoint := &inputEndpoint{}
	endpoint.SetInputData(inputAssembly{Keys: 0x2081, X: -200, Y: 1234})
	router := newRouter(endpoint)
	want := []byte{0x81, 0x20, 0, 0, 0x38, 0xff, 0xff, 0xff, 0xd2, 0x04, 0, 0}
	for _, path := range [][]byte{{1, 0}, {0x20, 4}, {0x34, 4}} {
		provider, err := router.Resolve(path)
		if err != nil {
			t.Fatalf("resolve path % x: %v", path, err)
		}
		if err := provider.IOWrite([]gologix.CIPItem{}); err != nil {
			t.Fatalf("ignore scanner output: %v", err)
		}
		got, err := provider.IORead()
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("path % x: got % x, err %v; want % x", path, got, err, want)
		}
	}
}
