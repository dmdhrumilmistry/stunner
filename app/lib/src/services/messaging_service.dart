import 'dart:async';
import 'dart:convert';

import 'package:path_provider/path_provider.dart';

import '../ffi/stunner_ffi.dart';
import 'chat_store.dart';

/// Bridges the Go live-messaging runtime (over FFI) to the [ChatStore].
///
/// On [start] it boots the runtime with a persistent account, then polls the
/// runtime's event queue on a timer and applies events to the store (incoming
/// messages, delivery status, presence). Outgoing sends are wired through
/// [ChatStore.onSend]. All FFI calls are non-blocking and run on the main
/// isolate (the runtime is a process-global owned there).
class MessagingService {
  MessagingService(this.core, this.store);

  final StunnerCore core;
  final ChatStore store;

  Timer? _timer;
  bool _started = false;

  /// This device's shareable contact URI (set after a successful [start]).
  String myContactUri = '';

  bool get started => _started;

  /// Boots the runtime under the per-platform app data dir, embedding
  /// [displayName] in the contact URI. Returns the URI on success, or an error.
  Future<({bool ok, String uri, String error})> start(String displayName) async {
    if (_started) return (ok: true, uri: myContactUri, error: '');
    if (!core.available) {
      return (ok: false, uri: '', error: 'Core library not loaded — messaging is unavailable.');
    }
    final String dir;
    try {
      dir = (await getApplicationSupportDirectory()).path;
    } on Object catch (e) {
      return (ok: false, uri: '', error: 'Could not open app data dir: $e');
    }
    final res = core.startRuntime(dir, displayName);
    if (res.error != null) {
      return (ok: false, uri: '', error: res.error!);
    }
    myContactUri = res.uri;
    _started = true;
    store.onSend = _send;
    store.onSendFile = (uri, path, msgId) {
      if (!_started || uri.isEmpty) {
        store.markFailed(msgId);
        return;
      }
      core.sendFile(uri, path, msgId);
    };
    store.onMarkRead = (uri) => core.markReadFor(uri);
    _timer = Timer.periodic(const Duration(milliseconds: 600), (_) => _drain());
    return (ok: true, uri: res.uri, error: '');
  }

  void _send(String peerUri, String text, String msgId) {
    if (!_started || peerUri.isEmpty) {
      store.markFailed(msgId);
      return;
    }
    core.sendMessage(peerUri, text, msgId);
  }

  void _drain() {
    if (!_started) return;
    final raw = core.pollEvents();
    if (raw.isEmpty || raw == '[]') return;
    final List<dynamic> events;
    try {
      events = jsonDecode(raw) as List<dynamic>;
    } on FormatException {
      return;
    }
    for (final item in events) {
      if (item is! Map<String, dynamic>) continue;
      final kind = item['kind'] as String? ?? '';
      final peerFp = item['peerFp'] as String? ?? '';
      if (kind == 'message') {
        store.receiveFromPeer(peerFp, item['peerUri'] as String? ?? '', item['text'] as String? ?? '');
      } else if (kind == 'file') {
        store.receiveFileFromPeer(peerFp, item['peerUri'] as String? ?? '',
            item['name'] as String? ?? 'file', item['path'] as String? ?? '');
      } else if (kind == 'sent') {
        store.markSent(item['msgId'] as String? ?? '');
      } else if (kind == 'sendFailed') {
        store.markFailed(item['msgId'] as String? ?? '');
      } else if (kind == 'presence') {
        store.setPresence(peerFp, item['online'] == true);
      } else if (kind == 'receipt') {
        final state = item['detail'] as String? ?? '';
        if (state == 'DELIVERED') {
          store.markDelivered(item['msgId'] as String? ?? '');
        } else if (state == 'READ') {
          store.markReadByPeer(peerFp);
        }
      }
    }
  }

  void stop() {
    _timer?.cancel();
    _timer = null;
    if (_started) core.stopRuntime();
    _started = false;
    store.onSend = null;
    store.onSendFile = null;
    store.onMarkRead = null;
  }
}
