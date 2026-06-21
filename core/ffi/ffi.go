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
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
	"github.com/dmdhrumilmistry/stunner/core/pkg/runtime"
)

// The messaging runtime is a process-global singleton driven over FFI. All
// exported calls are non-blocking: sends enqueue, and the Dart side polls
// StunnerPoll for incoming messages / status events.
var (
	rtMu sync.Mutex
	rt   *runtime.Runtime
)

//export StunnerStart
func StunnerStart(dataDir, handle *C.char) *C.char {
	rtMu.Lock()
	defer rtMu.Unlock()
	if rt == nil {
		r, err := runtime.Start(C.GoString(dataDir), C.GoString(handle))
		if err != nil {
			return C.CString("error: " + err.Error())
		}
		rt = r
	}
	// Returns "contactURI\tfingerprint".
	return C.CString(rt.MyURI() + "\t" + rt.Fingerprint())
}

//export StunnerSend
func StunnerSend(peerURI, text, msgID *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	r.Send(C.GoString(peerURI), C.GoString(text), C.GoString(msgID))
	return C.CString("ok")
}

//export StunnerSendTyping
func StunnerSendTyping(peerURI *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r != nil {
		r.SendTyping(C.GoString(peerURI))
	}
	return C.CString("ok")
}

//export StunnerGetSettings
func StunnerGetSettings() *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("{}")
	}
	return C.CString(r.Settings())
}

//export StunnerSetSettings
func StunnerSetSettings(jsonSettings *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	if err := r.SetSettings(C.GoString(jsonSettings)); err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString("ok")
}

//export StunnerSaveState
func StunnerSaveState(jsonState *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	if err := r.SaveState(C.GoString(jsonState)); err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString("ok")
}

//export StunnerLoadState
func StunnerLoadState() *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("")
	}
	return C.CString(r.LoadState())
}

//export StunnerSendFile
func StunnerSendFile(peerURI, path, msgID *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	r.SendFile(C.GoString(peerURI), C.GoString(path), C.GoString(msgID))
	return C.CString("ok")
}

//export StunnerMarkRead
func StunnerMarkRead(peerURI *C.char) *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	r.MarkRead(C.GoString(peerURI))
	return C.CString("ok")
}

//export StunnerPoll
func StunnerPoll() *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("[]")
	}
	b, err := json.Marshal(r.Poll())
	if err != nil {
		return C.CString("[]")
	}
	return C.CString(string(b))
}

//export StunnerMyURI
func StunnerMyURI() *C.char {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return C.CString("")
	}
	return C.CString(r.MyURI())
}

//export StunnerStop
func StunnerStop() *C.char {
	rtMu.Lock()
	r := rt
	rt = nil
	rtMu.Unlock()
	if r != nil {
		_ = r.Stop()
	}
	return C.CString("ok")
}

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

//export StunnerCheckSTUN
func StunnerCheckSTUN() *C.char {
	ok, addr, detail, err := core.CheckSTUN()
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	// Returns "ok|fail\t<reflexiveAddr>\t<detail>"; the Dart side splits on tabs.
	status := "fail"
	if ok {
		status = "ok"
	}
	return C.CString(status + "\t" + addr + "\t" + detail)
}

//export StunnerFree
func StunnerFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}

// main is required for buildmode=c-shared but is never executed.
func main() {}
