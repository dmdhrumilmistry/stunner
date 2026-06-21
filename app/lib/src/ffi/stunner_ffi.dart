// Dart FFI binding to the Stunner Go core (desktop c-shared library).
//
// The Go core is compiled with `go build -buildmode=c-shared` into
// libstunner.{so,dylib,dll} (see ../../../docs/ROADMAP.md). On mobile the core
// is bound via gomobile instead; that path is wired up in a later phase.
//
// This binding proves the boundary with Version/Ping/NewIdentityFingerprint.
// Strings returned by the core are heap-allocated in Go and freed via
// StunnerFree to avoid leaks.

import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';
import 'package:ffi/ffi.dart';

// C signatures.
typedef _VersionC = Pointer<Utf8> Function();
typedef _VersionDart = Pointer<Utf8> Function();

typedef _PingC = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _PingDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _FingerprintC = Pointer<Utf8> Function();
typedef _FingerprintDart = Pointer<Utf8> Function();

typedef _ContactURIC = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _ContactURIDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _SafetyC = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);
typedef _SafetyDart = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);

typedef _ValidateC = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _ValidateDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _CheckStunC = Pointer<Utf8> Function();
typedef _CheckStunDart = Pointer<Utf8> Function();

typedef _StartC = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);
typedef _StartDart = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);

typedef _SendC = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);
typedef _SendDart = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);

typedef _NoArgStrC = Pointer<Utf8> Function();
typedef _NoArgStrDart = Pointer<Utf8> Function();

typedef _FreeC = Void Function(Pointer<Utf8>);
typedef _FreeDart = void Function(Pointer<Utf8>);

/// Result of a STUN reachability probe (see [StunnerCore.checkStun]).
typedef StunResult = ({bool ok, String reflexiveAddr, String detail});

/// Thin wrapper over the native Stunner core.
///
/// Construct with [StunnerCore.open]. If the native library is not present
/// (e.g. running the UI shell before building the core), [available] is false
/// and the call methods return placeholder values so the app still runs.
class StunnerCore {
  StunnerCore._(this._lib);

  final DynamicLibrary? _lib;

  bool get available => _lib != null;

  late final _VersionDart _version =
      _lib!.lookupFunction<_VersionC, _VersionDart>('StunnerVersion');
  late final _PingDart _ping =
      _lib!.lookupFunction<_PingC, _PingDart>('StunnerPing');
  late final _FingerprintDart _fingerprint = _lib!
      .lookupFunction<_FingerprintC, _FingerprintDart>(
          'StunnerNewIdentityFingerprint');
  late final _ContactURIDart _contactURI = _lib!
      .lookupFunction<_ContactURIC, _ContactURIDart>('StunnerNewContactURI');
  late final _SafetyDart _safety =
      _lib!.lookupFunction<_SafetyC, _SafetyDart>('StunnerSafetyNumber');
  late final _ValidateDart _validate = _lib!
      .lookupFunction<_ValidateC, _ValidateDart>('StunnerValidateContactURI');
  late final _StartDart _start =
      _lib!.lookupFunction<_StartC, _StartDart>('StunnerStart');
  late final _SendDart _send =
      _lib!.lookupFunction<_SendC, _SendDart>('StunnerSend');
  late final _NoArgStrDart _poll =
      _lib!.lookupFunction<_NoArgStrC, _NoArgStrDart>('StunnerPoll');
  late final _NoArgStrDart _myUri =
      _lib!.lookupFunction<_NoArgStrC, _NoArgStrDart>('StunnerMyURI');
  late final _NoArgStrDart _stop =
      _lib!.lookupFunction<_NoArgStrC, _NoArgStrDart>('StunnerStop');
  late final _FreeDart _free =
      _lib!.lookupFunction<_FreeC, _FreeDart>('StunnerFree');

  /// Loads the native library for the current desktop platform.
  static StunnerCore open() {
    try {
      return StunnerCore._(_load());
    } on Object {
      // Library not built yet — run in degraded mode.
      return StunnerCore._(null);
    }
  }

  static DynamicLibrary _load() {
    // Android: the core is bundled as jniLibs/<abi>/libstunner.so and resolved
    // by name. iOS would require static linking (not yet wired).
    if (Platform.isAndroid) return DynamicLibrary.open('libstunner.so');
    if (Platform.isIOS) return DynamicLibrary.process();

    // Desktop: try paths relative to the executable / app bundle first (where
    // the release packages the library), then fall back to the bare name.
    final exeDir = File(Platform.resolvedExecutable).parent.path;
    final sep = Platform.pathSeparator;
    final candidates = <String>[];
    if (Platform.isWindows) {
      candidates.addAll(['$exeDir${sep}stunner.dll', 'stunner.dll']);
    } else if (Platform.isMacOS) {
      candidates.addAll([
        '$exeDir$sep..${sep}Frameworks${sep}libstunner.dylib',
        '$exeDir${sep}libstunner.dylib',
        'libstunner.dylib',
      ]);
    } else {
      candidates.addAll([
        '$exeDir${sep}lib${sep}libstunner.so',
        '$exeDir${sep}libstunner.so',
        'libstunner.so',
      ]);
    }
    for (final path in candidates) {
      try {
        return DynamicLibrary.open(path);
      } on Object {
        // try next candidate
      }
    }
    throw Exception('libstunner not found (looked in: ${candidates.join(", ")})');
  }

  String version() {
    if (!available) return 'core unavailable (build libstunner)';
    final ptr = _version();
    final s = ptr.toDartString();
    _free(ptr);
    return s;
  }

  String ping(String msg) {
    if (!available) return 'core unavailable';
    final arg = msg.toNativeUtf8();
    try {
      final ptr = _ping(arg);
      final s = ptr.toDartString();
      _free(ptr);
      return s;
    } finally {
      malloc.free(arg);
    }
  }

  String newIdentityFingerprint() {
    if (!available) return 'core unavailable';
    final ptr = _fingerprint();
    final s = ptr.toDartString();
    _free(ptr);
    return s;
  }

  /// Generates a fresh identity and returns its shareable `stunner:contact` URI
  /// (render this as a QR code). Ephemeral until the persistent account is
  /// exposed over FFI.
  String newContactURI(String handle) {
    if (!available) return 'stunner:contact?n=$handle';
    final arg = handle.toNativeUtf8();
    try {
      final ptr = _contactURI(arg);
      final s = ptr.toDartString();
      _free(ptr);
      return s;
    } finally {
      malloc.free(arg);
    }
  }

  /// Computes the verification safety number between two contacts, each given as
  /// a `stunner:contact` URI (e.g. your own and one scanned from a QR code).
  String safetyNumber(String myContactURI, String peerContactURI) {
    if (!available) return 'core unavailable';
    final a = myContactURI.toNativeUtf8();
    final b = peerContactURI.toNativeUtf8();
    try {
      final ptr = _safety(a, b);
      final s = ptr.toDartString();
      _free(ptr);
      return s;
    } finally {
      malloc.free(a);
      malloc.free(b);
    }
  }

  /// Validates a scanned contact URI, returning (handle, fingerprint) or
  /// throwing if the core reports an error.
  ({String handle, String fingerprint}) validateContactURI(String uri) {
    if (!available) return (handle: '', fingerprint: 'core unavailable');
    final arg = uri.toNativeUtf8();
    try {
      final ptr = _validate(arg);
      final s = ptr.toDartString();
      _free(ptr);
      if (s.startsWith('error: ')) {
        throw FormatException(s.substring(7));
      }
      final parts = s.split('\t');
      return (handle: parts.first, fingerprint: parts.length > 1 ? parts[1] : '');
    } finally {
      malloc.free(arg);
    }
  }

  /// Probes the default STUN servers and reports whether a public address could
  /// be discovered. Runs in a background isolate so the (network-bound) probe
  /// never blocks the UI thread.
  Future<StunResult> checkStun() async {
    if (!available) {
      return (ok: false, reflexiveAddr: '', detail: 'core unavailable (build libstunner)');
    }
    try {
      return await Isolate.run(_checkStunWorker);
    } on Object catch (e) {
      return (ok: false, reflexiveAddr: '', detail: 'STUN check failed: $e');
    }
  }

  // --- live messaging runtime ---
  //
  // These drive the process-global Go runtime and must all run on the isolate
  // that loaded the library (the main isolate). They are non-blocking: sends
  // enqueue, and [pollEvents] drains queued incoming-message / status events.

  /// Starts the messaging runtime with a persistent account under [dataDir],
  /// embedding [handle] in the shareable contact URI. Returns the URI +
  /// fingerprint, or an error string.
  ({String uri, String fingerprint, String? error}) startRuntime(String dataDir, String handle) {
    if (!available) return (uri: '', fingerprint: '', error: 'core unavailable (build libstunner)');
    final a = dataDir.toNativeUtf8();
    final b = handle.toNativeUtf8();
    try {
      final ptr = _start(a, b);
      final s = ptr.toDartString();
      _free(ptr);
      if (s.startsWith('error: ')) return (uri: '', fingerprint: '', error: s.substring(7));
      final parts = s.split('\t');
      return (uri: parts.first, fingerprint: parts.length > 1 ? parts[1] : '', error: null);
    } finally {
      malloc.free(a);
      malloc.free(b);
    }
  }

  /// Enqueues a text message to the peer at [peerUri]. Returns immediately.
  void sendMessage(String peerUri, String text, String msgId) {
    if (!available) return;
    final a = peerUri.toNativeUtf8();
    final b = text.toNativeUtf8();
    final c = msgId.toNativeUtf8();
    try {
      _free(_send(a, b, c));
    } finally {
      malloc.free(a);
      malloc.free(b);
      malloc.free(c);
    }
  }

  /// Drains pending runtime events as a JSON array string ("[]" if none).
  String pollEvents() {
    if (!available) return '[]';
    final ptr = _poll();
    final s = ptr.toDartString();
    _free(ptr);
    return s;
  }

  /// The account's shareable contact URI (empty until the runtime is started).
  String runtimeUri() {
    if (!available) return '';
    final ptr = _myUri();
    final s = ptr.toDartString();
    _free(ptr);
    return s;
  }

  /// Stops the runtime.
  void stopRuntime() {
    if (!available) return;
    _free(_stop());
  }
}

/// Runs the STUN probe inside a fresh isolate: re-opens the native library
/// (isolates do not share FFI handles), invokes the export, and parses the
/// `status\treflexiveAddr\tdetail` payload.
StunResult _checkStunWorker() {
  final lib = StunnerCore._load();
  final check = lib.lookupFunction<_CheckStunC, _CheckStunDart>('StunnerCheckSTUN');
  final free = lib.lookupFunction<_FreeC, _FreeDart>('StunnerFree');
  final ptr = check();
  final s = ptr.toDartString();
  free(ptr);
  if (s.startsWith('error: ')) {
    return (ok: false, reflexiveAddr: '', detail: s.substring(7));
  }
  final parts = s.split('\t');
  return (
    ok: parts.isNotEmpty && parts[0] == 'ok',
    reflexiveAddr: parts.length > 1 ? parts[1] : '',
    detail: parts.length > 2 ? parts[2] : '',
  );
}
