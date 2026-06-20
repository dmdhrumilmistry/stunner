// Package main builds the desktop C-shared library consumed by the Flutter app
// over dart:ffi on macOS, Windows, and Linux.
//
// Build with:
//
//	go build -buildmode=c-shared -o ../app/native/libstunner.so ./ffi
//
// (.dylib on macOS, .dll on Windows). This produces libstunner.h with the
// exported symbols below. The skeleton exports a Version/Ping smoke test plus a
// fingerprint generator; richer APIs are added per roadmap phase.
//
// Strings returned to C are heap-allocated with C.CString and must be freed by
// the caller via StunnerFree to avoid leaks. The Dart binding wraps this.
//
//go:build cgo

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
)

//export StunnerVersion
func StunnerVersion() *C.char {
	return C.CString(core.VersionString())
}

//export StunnerPing
func StunnerPing(msg *C.char) *C.char {
	return C.CString("pong: " + C.GoString(msg))
}

//export StunnerNewIdentityFingerprint
func StunnerNewIdentityFingerprint() *C.char {
	fp, err := core.NewIdentityFingerprint()
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(fp)
}

//export StunnerFree
func StunnerFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}

// main is required for buildmode=c-shared but is never executed.
func main() {}
