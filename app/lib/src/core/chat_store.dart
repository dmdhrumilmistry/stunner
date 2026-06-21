import 'dart:async';

import 'package:flutter/foundation.dart';

import '../ffi/stunner_ffi.dart';
import '../models/chat.dart';
import '../services/emoji.dart';
import 'event_pump.dart';

/// Holds conversations and messages, fed by the Go core over FFI. A plain
/// [ChangeNotifier] (no state-management package) consistent with the app's
/// existing style.
///
/// Conversations are keyed by the peer's identity fingerprint — the stable 1:1
/// id both devices agree on — so inbound messages route to the right thread,
/// and messages from peers not yet in contacts auto-create a chat.
class ChatStore extends ChangeNotifier {
  ChatStore(this._core, this._events);

  final StunnerCore _core;
  final CoreEventSource _events;
  StreamSubscription<CoreEvent>? _sub;

  final Map<String, Chat> _chats = {}; // peerFp -> chat
  final Map<String, List<Message>> _messages = {}; // peerFp -> messages

  String? lastError;

  List<Chat> get chats {
    final list = _chats.values.toList();
    list.sort((a, b) => a.displayName.compareTo(b.displayName));
    return list;
  }

  List<Message> messagesFor(String convId) =>
      List.unmodifiable(_messages[convId] ?? const []);

  Chat? chatFor(String convId) => _chats[convId];

  /// Subscribes to core events and starts the event source.
  void bootstrap() {
    _sub = _events.events.listen(_onEvent);
    _events.start();
  }

  /// Pauses/resumes event polling (driven by app lifecycle to save battery).
  void pausePolling() => _events.pause();
  void resumePolling() => _events.resume();

  /// Connects to a peer from a scanned/pasted contact URI, creating the chat.
  /// Returns the conversation id (peer fingerprint).
  String connect(String contactUri) {
    final info = _core.validateContactURI(contactUri); // throws if malformed
    final peerFp = _core.connect(contactUri); // throws on failure
    final chat = _chats.putIfAbsent(
      peerFp,
      () => Chat(convId: peerFp, displayName: ''),
    );
    chat.peerFingerprint = peerFp;
    chat.peerContactURI = contactUri;
    chat.isContact = true;
    chat.connected = true;
    if (info.handle.isNotEmpty) chat.displayName = info.handle;
    if (chat.displayName.isEmpty) chat.displayName = _shortFp(peerFp);
    notifyListeners();
    return peerFp;
  }

  /// Sends [text] to the conversation [convId] (peer fingerprint), optimistically
  /// appending it locally.
  void send(String convId, String text) {
    final expanded = expandShortcodes(text.trim());
    if (expanded.isEmpty) return;
    String state = 'SENT';
    String id;
    try {
      id = _core.sendText(convId, convId, expanded);
    } on Object catch (e) {
      id = 'local-${DateTime.now().microsecondsSinceEpoch}';
      state = 'FAILED';
      lastError = '$e';
    }
    _appendMessage(convId, Message(id: id, text: expanded, fromMe: true, state: state));
    final chat = _chats[convId];
    if (chat != null) chat.lastMessage = expanded;
    notifyListeners();
  }

  /// Clears the unread badge for a conversation.
  void markRead(String convId) {
    final chat = _chats[convId];
    if (chat != null && chat.unread != 0) {
      chat.unread = 0;
      notifyListeners();
    }
  }

  void _onEvent(CoreEvent ev) {
    switch (ev) {
      case final IncomingMessage m:
        final chat = _ensureChat(m.peerFp);
        _appendMessage(m.peerFp, Message(id: m.msgId, text: m.text, fromMe: false));
        chat.lastMessage = m.text;
        chat.unread += 1;
      case final PeerConnected c:
        _ensureChat(c.peerFp).connected = true;
      case final PeerDisconnected d:
        final chat = _chats[d.peerFp];
        if (chat != null) chat.connected = false;
      case final CoreError e:
        lastError = e.message;
    }
    notifyListeners();
  }

  /// Returns the chat for a peer fingerprint, creating an "Unknown" one for a
  /// non-contact sender.
  Chat _ensureChat(String peerFp) =>
      _chats.putIfAbsent(peerFp, () => Chat.unknownFrom(peerFp, peerFp));

  void _appendMessage(String convId, Message m) {
    final list = _messages.putIfAbsent(convId, () => <Message>[]);
    // Dedupe by message id (poll may surface the same event once).
    if (m.id.isNotEmpty && list.any((x) => x.id == m.id)) return;
    list.add(m);
  }

  String _shortFp(String fp) {
    final s = fp.replaceAll(' ', '');
    return s.length >= 6 ? s.substring(0, 6) : s;
  }

  @override
  void dispose() {
    _sub?.cancel();
    _events.dispose();
    super.dispose();
  }
}
