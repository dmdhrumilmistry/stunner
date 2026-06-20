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

//export StunnerNewContactURI
func StunnerNewContactURI(handle *C.char) *C.char {
	uri, err := core.NewContactURI(C.GoString(handle))
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(uri)
}

//export StunnerSafetyNumber
func StunnerSafetyNumber(myContactURI, peerContactURI *C.char) *C.char {
	sn, err := core.SafetyNumber(C.GoString(myContactURI), C.GoString(peerContactURI))
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(sn)
}

//export StunnerValidateContactURI
func StunnerValidateContactURI(uri *C.char) *C.char {
	handle, fp, err := core.ValidateContactURI(C.GoString(uri))
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	// Returns "handle\tfingerprint"; the Dart side splits on the tab.
	return C.CString(handle + "\t" + fp)
}

//export StunnerFree
func StunnerFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}

// main is required for buildmode=c-shared but is never executed.
func main() {}
