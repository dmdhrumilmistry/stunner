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
	"encoding/hex"
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
	"github.com/dmdhrumilmistry/stunner/core/pkg/session"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
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

// --- stateful runtime --------------------------------------------------------
//
// The c-shared library keeps the Go runtime resident across calls, so a single
// process-global *session.Runtime (guarded by rtMu) backs the live messaging
// path. Inbound messages are delivered by polling: the app calls StunnerPoll on
// a timer and receives a JSON array of events. Result strings use the same
// "error: ..." convention as the helpers above and must be freed with
// StunnerFree.

var (
	rtMu sync.Mutex
	rt   *session.Runtime
)

func currentRuntime() *session.Runtime {
	rtMu.Lock()
	defer rtMu.Unlock()
	return rt
}

// StunnerStart boots the runtime for the account at accountDir. key is 64 hex
// chars (32 bytes) from the OS secure store; iceServersJSON is a JSON array of
// {urls,username,credential} (empty string uses the built-in STUN defaults).
// Returns "" on success or "error: ...".
//
//export StunnerStart
func StunnerStart(accountDir, keyHex, iceServersJSON *C.char) *C.char {
	rtMu.Lock()
	defer rtMu.Unlock()
	if rt != nil {
		return C.CString("error: runtime already started")
	}
	key, err := hex.DecodeString(C.GoString(keyHex))
	if err != nil {
		return C.CString("error: invalid key hex: " + err.Error())
	}
	var iceServers []settings.ICEServer
	if js := C.GoString(iceServersJSON); js != "" {
		if err := json.Unmarshal([]byte(js), &iceServers); err != nil {
			return C.CString("error: invalid iceServers json: " + err.Error())
		}
	}
	r, err := session.Start(session.Config{
		AccountDir: C.GoString(accountDir),
		Key:        key,
		Settings:   settings.Settings{ICEServers: iceServers},
		Rendezvous: session.DefaultRendezvous,
	})
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	rt = r
	return C.CString("")
}

// StunnerConnect discovers and dials the peer named by a scanned
// "stunner:contact" URI, returning the peer fingerprint or "error: ...".
//
//export StunnerConnect
func StunnerConnect(contactURI *C.char) *C.char {
	r := currentRuntime()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	fp, err := r.Connect(C.GoString(contactURI))
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(fp)
}

// StunnerSendText sends text to the peer identified by peerFP within convID,
// returning the message id or "error: ...".
//
//export StunnerSendText
func StunnerSendText(convID, peerFP, text *C.char) *C.char {
	r := currentRuntime()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	id, err := r.SendText(C.GoString(convID), C.GoString(peerFP), C.GoString(text))
	if err != nil {
		return C.CString("error: " + err.Error())
	}
	return C.CString(id)
}

// StunnerPoll drains and returns all buffered events as a JSON array (e.g.
// [{"kind":"message","convId":"..","peerFp":"..","text":"..","msgId":".."}]).
// Returns "[]" when there is nothing pending or the runtime is not started.
//
//export StunnerPoll
func StunnerPoll() *C.char {
	r := currentRuntime()
	if r == nil {
		return C.CString("[]")
	}
	events := r.DrainEvents()
	if len(events) == 0 {
		return C.CString("[]")
	}
	b, err := json.Marshal(events)
	if err != nil {
		return C.CString("[]")
	}
	return C.CString(string(b))
}

// StunnerMyContactURI returns the started account's persistent contact URI for
// the given handle (render as a QR code), or "error: ...". Unlike
// StunnerNewContactURI this reflects the identity the runtime actually uses.
//
//export StunnerMyContactURI
func StunnerMyContactURI(handle *C.char) *C.char {
	r := currentRuntime()
	if r == nil {
		return C.CString("error: runtime not started")
	}
	return C.CString(r.ContactURI(C.GoString(handle)))
}

// StunnerStop tears the runtime down (closes links, transport, signaler, store).
//
//export StunnerStop
func StunnerStop() {
	rtMu.Lock()
	r := rt
	rt = nil
	rtMu.Unlock()
	if r != nil {
		_ = r.Stop()
	}
}

// main is required for buildmode=c-shared but is never executed.
func main() {}
