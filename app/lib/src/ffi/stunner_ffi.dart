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

typedef _FreeC = Void Function(Pointer<Utf8>);
typedef _FreeDart = void Function(Pointer<Utf8>);

// Stateful runtime (live messaging path).
typedef _StartC = Pointer<Utf8> Function(
    Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);
typedef _StartDart = Pointer<Utf8> Function(
    Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);

typedef _ConnectC = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _ConnectDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _SendTextC = Pointer<Utf8> Function(
    Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);
typedef _SendTextDart = Pointer<Utf8> Function(
    Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);

typedef _PollC = Pointer<Utf8> Function();
typedef _PollDart = Pointer<Utf8> Function();

typedef _MyContactURIC = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _MyContactURIDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _StopC = Void Function();
typedef _StopDart = void Function();

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
  late final _FreeDart _free =
      _lib!.lookupFunction<_FreeC, _FreeDart>('StunnerFree');
  late final _StartDart _start =
      _lib!.lookupFunction<_StartC, _StartDart>('StunnerStart');
  late final _ConnectDart _connect =
      _lib!.lookupFunction<_ConnectC, _ConnectDart>('StunnerConnect');
  late final _SendTextDart _sendText =
      _lib!.lookupFunction<_SendTextC, _SendTextDart>('StunnerSendText');
  late final _PollDart _poll =
      _lib!.lookupFunction<_PollC, _PollDart>('StunnerPoll');
  late final _MyContactURIDart _myContactURI = _lib!
      .lookupFunction<_MyContactURIC, _MyContactURIDart>('StunnerMyContactURI');
  late final _StopDart _stop =
      _lib!.lookupFunction<_StopC, _StopDart>('StunnerStop');

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

  // --- stateful runtime (live messaging) -------------------------------------

  /// Reads a native result string, frees it, and throws [StunnerException] if it
  /// is an "error: ..." sentinel.
  String _unwrap(Pointer<Utf8> ptr) {
    final s = ptr.toDartString();
    _free(ptr);
    if (s.startsWith('error: ')) {
      throw StunnerException(s.substring(7));
    }
    return s;
  }

  /// Starts the runtime: opens the account at [accountDir] (creating it on first
  /// run), brings up signaling + transport, and begins accepting peers.
  ///
  /// [keyHex] is the 32-byte vault key as 64 hex chars (from the OS secure
  /// store). [iceServersJson] is a JSON array of `{urls,username,credential}`
  /// (empty string uses the built-in STUN defaults). No-op in degraded mode.
  void start({
    required String accountDir,
    required String keyHex,
    String iceServersJson = '',
  }) {
    if (!available) return;
    final dir = accountDir.toNativeUtf8();
    final key = keyHex.toNativeUtf8();
    final ice = iceServersJson.toNativeUtf8();
    try {
      _unwrap(_start(dir, key, ice));
    } finally {
      malloc.free(dir);
      malloc.free(key);
      malloc.free(ice);
    }
  }

  /// Discovers and dials the peer named by a scanned `stunner:contact` URI,
  /// returning the peer fingerprint. Throws on failure.
  String connect(String contactUri) {
    if (!available) throw const StunnerException('core unavailable');
    final arg = contactUri.toNativeUtf8();
    try {
      return _unwrap(_connect(arg));
    } finally {
      malloc.free(arg);
    }
  }

  /// Sends [text] to the peer [peerFp] within [convId], returning the message
  /// id. Throws on failure.
  String sendText(String convId, String peerFp, String text) {
    if (!available) throw const StunnerException('core unavailable');
    final c = convId.toNativeUtf8();
    final p = peerFp.toNativeUtf8();
    final t = text.toNativeUtf8();
    try {
      return _unwrap(_sendText(c, p, t));
    } finally {
      malloc.free(c);
      malloc.free(p);
      malloc.free(t);
    }
  }

  /// Drains and returns buffered events as a JSON array string (e.g.
  /// `[{"kind":"message","convId":"..","peerFp":"..","text":"..","msgId":".."}]`).
  /// Returns `"[]"` when nothing is pending or the core is unavailable.
  String pollEventsJson() {
    if (!available) return '[]';
    final ptr = _poll();
    final s = ptr.toDartString();
    _free(ptr);
    return s;
  }

  /// The started account's persistent contact URI (render as a QR code). Throws
  /// if the runtime is not started.
  String myContactURI(String handle) {
    if (!available) return 'stunner:contact?n=$handle';
    final arg = handle.toNativeUtf8();
    try {
      return _unwrap(_myContactURI(arg));
    } finally {
      malloc.free(arg);
    }
  }

  /// Tears the runtime down. Safe to call when not started.
  void stop() {
    if (!available) return;
    _stop();
  }
}

/// Thrown when the native core returns an "error: ..." result.
class StunnerException implements Exception {
  const StunnerException(this.message);
  final String message;
  @override
  String toString() => 'StunnerException: $message';
}
