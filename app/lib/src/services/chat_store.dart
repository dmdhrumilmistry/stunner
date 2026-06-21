import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/chat.dart';
import 'app_state.dart' show Prefs;
import 'notification_service.dart';

/// Result of a peer connectivity test ("Test connection").
class ConnectionDiagnostic {
  const ConnectionDiagnostic({required this.ok, required this.message}) : testing = false;
  const ConnectionDiagnostic.testing()
      : ok = false,
        message = 'Testing connection…',
        testing = true;

  final bool ok;
  final String message;
  final bool testing;
}

/// In-memory chats + contacts shared by every screen, so the list and the
/// conversation always agree. Local demo store: the live two-device path
/// (WebRTC over STUN/TURN via the Go core) is wired over FFI in a later step.
///
/// The app starts empty — add a contact to begin. Incoming messages flow through
/// [receiveFromPeer], which raises an in-app notification via [notifications].
class ChatStore extends ChangeNotifier {
  ChatStore({this.notifications});

  /// Optional sink for in-app live notifications on incoming messages.
  final NotificationService? notifications;

  /// Outbound hook wired by the messaging runtime: (peerContactUri, text, msgId).
  /// When null (runtime not started) sends are marked failed.
  void Function(String peerUri, String text, String msgId)? onSend;

  /// Hook wired by the runtime to send a read receipt for a peer's contact URI.
  void Function(String peerUri)? onMarkRead;

  /// Outbound file hook: (peerContactUri, localPath, msgId).
  void Function(String peerUri, String path, String msgId)? onSendFile;

  /// Outbound typing hook: (peerContactUri). Gated by [Prefs.typingIndicators].
  void Function(String peerUri)? onTyping;

  /// Connectivity-test hook: (peerContactUri). Result returns via [applyDiagnostic].
  void Function(String peerUri)? onDiagnose;

  /// Last connectivity-test result per contact id (fingerprint).
  final Map<String, ConnectionDiagnostic> _diagnostics = {};

  ConnectionDiagnostic? diagnosticFor(String contactId) => _diagnostics[contactId];

  /// Starts a connectivity test for a chat's peer (result via [applyDiagnostic]).
  void testConnection(String chatId) {
    final chat = maybeChat(chatId);
    if (chat == null) return;
    final contact = contactForChat(chat);
    final uri = contact?.code ?? '';
    if (contact == null || uri.isEmpty) return;
    _diagnostics[contact.id] = const ConnectionDiagnostic.testing();
    notifyListeners();
    if (onDiagnose != null) {
      onDiagnose!(uri);
    } else {
      _diagnostics[contact.id] =
          const ConnectionDiagnostic(ok: false, message: 'Messaging core not available.');
      notifyListeners();
    }
  }

  /// Applies a diagnostic result from the runtime (by peer fingerprint).
  void applyDiagnostic(String peerFingerprint, bool ok, String message) {
    if (peerFingerprint.isEmpty) return;
    _diagnostics[peerFingerprint] = ConnectionDiagnostic(ok: ok, message: message);
    notifyListeners();
  }

  /// Dismisses a shown diagnostic for a contact.
  void clearDiagnostic(String contactId) {
    if (_diagnostics.remove(contactId) != null) notifyListeners();
  }

  /// User preferences (set by the messaging service); gates receipts/typing/
  /// notification previews when present.
  Prefs? prefs;

  /// peerFingerprint -> time until which they're considered "typing".
  final Map<String, DateTime> _typingUntil = {};
  Timer? _typingTimer;

  /// Whether the contact (by id == fingerprint) is currently typing.
  bool isTyping(String contactId) {
    final until = _typingUntil[contactId];
    return until != null && DateTime.now().isBefore(until);
  }

  /// Records an inbound typing indicator (auto-expires after a few seconds).
  void receiveTyping(String peerFingerprint) {
    _typingUntil[peerFingerprint] = DateTime.now().add(const Duration(seconds: 6));
    notifyListeners();
    _typingTimer?.cancel();
    _typingTimer = Timer(const Duration(seconds: 6), notifyListeners);
  }

  /// Sends a typing indicator for the chat (no-op if disabled in prefs).
  void sendTyping(String chatId) {
    if (prefs != null && !prefs!.typingIndicators) return;
    final chat = maybeChat(chatId);
    if (chat == null) return;
    final uri = contactForChat(chat)?.code ?? '';
    if (onTyping != null && uri.isNotEmpty) onTyping!(uri);
  }

  final List<Contact> _contacts = [];
  final List<Chat> _chats = [];

  List<Contact> get contacts => List.unmodifiable(_contacts);
  List<Chat> get chats => List.unmodifiable(_chats);

  /// Serializes contacts + conversations for encrypted persistence.
  Map<String, dynamic> toMap() => {
        'contacts': _contacts.map((c) => c.toMap()).toList(),
        'chats': _chats.map((c) => c.toMap()).toList(),
      };

  /// Replaces all state from a previously serialized map (used on startup).
  /// Does not notify — the caller restores the whole app then refreshes once.
  void restoreFromMap(Map<String, dynamic> m) {
    _contacts
      ..clear()
      ..addAll(((m['contacts'] as List?) ?? const [])
          .map((e) => Contact.fromMap((e as Map).cast<String, dynamic>())));
    _chats
      ..clear()
      ..addAll(((m['chats'] as List?) ?? const [])
          .map((e) => Chat.fromMap((e as Map).cast<String, dynamic>())));
  }

  Chat chatById(String id) => _chats.firstWhere((c) => c.id == id);
  Chat? maybeChat(String id) {
    for (final c in _chats) {
      if (c.id == id) return c;
    }
    return null;
  }

  Contact? contactById(String id) {
    for (final c in _contacts) {
      if (c.id == id) return c;
    }
    return null;
  }

  /// The contact a chat is with, if still known.
  Contact? contactForChat(Chat chat) => contactById(chat.contactId);

  // --- contacts ---

  /// Adds (or updates) a contact and returns it. Deduplicated by fingerprint
  /// when present, otherwise by name. Presence is unknown until the runtime
  /// reports it, so new contacts start offline.
  Contact addContact({
    required String name,
    String code = '',
    String fingerprint = '',
    String role = '',
  }) {
    final cleanName = name.trim().isEmpty ? 'Unnamed' : name.trim();
    final existing = _contacts.where((c) =>
        (fingerprint.isNotEmpty && c.fingerprint == fingerprint) ||
        (fingerprint.isEmpty && c.name == cleanName));
    if (existing.isNotEmpty) {
      final c = existing.first;
      c.name = cleanName;
      if (code.isNotEmpty) c.code = code;
      notifyListeners();
      return c;
    }
    final contact = Contact(
      id: fingerprint.isNotEmpty ? fingerprint : 'k-${DateTime.now().microsecondsSinceEpoch}',
      name: cleanName,
      code: code,
      fingerprint: fingerprint,
      role: role,
    );
    _contacts.add(contact);
    notifyListeners();
    return contact;
  }

  void deleteContact(String contactId) {
    _contacts.removeWhere((c) => c.id == contactId);
    // Also drop conversations with that contact.
    _chats.removeWhere((c) => c.contactId == contactId);
    notifyListeners();
  }

  void toggleMute(String contactId) {
    final c = contactById(contactId);
    if (c == null) return;
    c.muted = !c.muted;
    notifyListeners();
  }

  void toggleBlock(String contactId) {
    final c = contactById(contactId);
    if (c == null) return;
    c.blocked = !c.blocked;
    notifyListeners();
  }

  // --- chats ---

  /// Opens (or creates) the conversation with a contact and returns its id.
  String startChatWith(Contact contact) {
    final existing = _chats.where((c) => c.contactId == contact.id);
    if (existing.isNotEmpty) {
      final chat = existing.first;
      _moveToTop(chat);
      notifyListeners();
      return chat.id;
    }
    final chat = Chat(
      id: 'c${DateTime.now().microsecondsSinceEpoch}',
      contactId: contact.id,
      name: contact.name,
    );
    _chats.insert(0, chat);
    notifyListeners();
    return chat.id;
  }

  void deleteChat(String chatId) {
    _chats.removeWhere((c) => c.id == chatId);
    notifyListeners();
  }

  void deleteMessage(String chatId, String messageId) {
    maybeChat(chatId)?.messages.removeWhere((m) => m.id == messageId);
    notifyListeners();
  }

  /// Toggles the local user's reaction to a message (0/1 in this demo store).
  void toggleReaction(String chatId, String messageId, String emoji) {
    final chat = maybeChat(chatId);
    if (chat == null) return;
    for (final m in chat.messages) {
      if (m.id == messageId) {
        if ((m.reactions[emoji] ?? 0) > 0) {
          m.reactions.remove(emoji);
        } else {
          m.reactions[emoji] = 1;
        }
        notifyListeners();
        return;
      }
    }
  }

  /// Sends a text over the live runtime via [onSend]. The message starts as
  /// "sending"; the runtime later reports "sent" or "failed" (see [markSent] /
  /// [markFailed]). If no runtime is wired or the peer has no contact ID, the
  /// message is marked failed immediately.
  void sendText(String chatId, String text) {
    final body = text.trim();
    if (body.isEmpty) return;
    final chat = chatById(chatId);
    final contact = contactForChat(chat);
    final msg = Message(
      id: 'local-${DateTime.now().microsecondsSinceEpoch}',
      text: body,
      fromMe: true,
      status: DeliveryStatus.sending,
    );
    chat.messages.add(msg);
    _moveToTop(chat);
    notifyListeners();

    final uri = contact?.code ?? '';
    if (onSend != null && uri.isNotEmpty) {
      onSend!(uri, body, msg.id);
    } else {
      msg.status = DeliveryStatus.failed;
      notifyListeners();
    }
  }

  /// Sends a file (by local [path]) over the runtime via [onSendFile].
  void sendFile(String chatId, String path) {
    final name = path.split(RegExp(r'[/\\]')).last;
    final chat = chatById(chatId);
    final contact = contactForChat(chat);
    final msg = Message(
      id: 'localf-${DateTime.now().microsecondsSinceEpoch}',
      text: '',
      fromMe: true,
      status: DeliveryStatus.sending,
      fileName: name,
      filePath: path,
    );
    chat.messages.add(msg);
    _moveToTop(chat);
    notifyListeners();

    final uri = contact?.code ?? '';
    if (onSendFile != null && uri.isNotEmpty) {
      onSendFile!(uri, path, msg.id);
    } else {
      _setStatus(msg.id, DeliveryStatus.failed);
    }
  }

  /// Delivers an incoming file from a peer into its conversation.
  void receiveFileFromPeer(String peerFingerprint, String peerUri, String name, String path) {
    if (peerFingerprint.isEmpty) return;
    var contact = contactById(peerFingerprint);
    if (contact == null) {
      final tag = peerFingerprint.length <= 5 ? peerFingerprint : peerFingerprint.substring(0, 5);
      contact = addContact(name: 'Contact $tag', code: peerUri, fingerprint: peerFingerprint);
    }
    if (contact.code.isEmpty && peerUri.isNotEmpty) contact.code = peerUri;

    final chatId = startChatWith(contact);
    final chat = chatById(chatId);
    chat.messages.add(Message(
      id: 'inf-${DateTime.now().microsecondsSinceEpoch}',
      text: '',
      fromMe: false,
      fileName: name,
      filePath: path,
    ));
    chat.unread += 1;
    _typingUntil.remove(peerFingerprint);
    _moveToTop(chat);
    notifyListeners();
    if (!contact.muted) {
      final preview = (prefs?.notifPreview ?? true) ? '📎 $name' : 'New message';
      notifications?.push(title: chat.name, body: preview, chatId: chatId);
    }
  }

  /// Marks an outgoing message (by id) as sent (queued to the peer).
  void markSent(String msgId) => _setStatus(msgId, DeliveryStatus.sent);

  /// Marks an outgoing message (by id) as failed.
  void markFailed(String msgId) => _setStatus(msgId, DeliveryStatus.failed);

  /// Marks an outgoing message (by id) as delivered to the peer.
  void markDelivered(String msgId) {
    // Don't regress a message already marked read.
    for (final c in _chats) {
      for (final m in c.messages) {
        if (m.id == msgId) {
          if (m.status != DeliveryStatus.read) {
            m.status = DeliveryStatus.delivered;
            notifyListeners();
          }
          return;
        }
      }
    }
  }

  /// Marks all of our sent messages in the peer's chat as read (read receipt).
  void markReadByPeer(String peerFingerprint) {
    for (final chat in _chats) {
      if (chat.contactId != peerFingerprint) continue;
      var changed = false;
      for (final m in chat.messages) {
        if (m.fromMe && m.status != DeliveryStatus.read) {
          m.status = DeliveryStatus.read;
          changed = true;
        }
      }
      if (changed) notifyListeners();
      return;
    }
  }

  void _setStatus(String msgId, DeliveryStatus status) {
    for (final c in _chats) {
      for (final m in c.messages) {
        if (m.id == msgId) {
          m.status = status;
          notifyListeners();
          return;
        }
      }
    }
  }

  /// Updates a contact's presence from a runtime event.
  void setPresence(String peerFingerprint, bool online) {
    final c = contactById(peerFingerprint);
    if (c != null && c.online != online) {
      c.online = online;
      notifyListeners();
    }
  }

  /// Delivers an incoming message from a peer (identified by fingerprint), into
  /// its conversation — creating the contact/chat if this peer is new — and
  /// raises an in-app notification. [peerUri] (when present) makes an
  /// inbound-only peer repliable.
  void receiveFromPeer(String peerFingerprint, String peerUri, String text) {
    if (peerFingerprint.isEmpty) return;
    var contact = contactById(peerFingerprint);
    if (contact == null) {
      final tag = peerFingerprint.length <= 5 ? peerFingerprint : peerFingerprint.substring(0, 5);
      contact = addContact(name: 'Contact $tag', code: peerUri, fingerprint: peerFingerprint);
    }
    if (contact.code.isEmpty && peerUri.isNotEmpty) contact.code = peerUri;

    final chatId = startChatWith(contact);
    final chat = chatById(chatId);
    chat.messages.add(Message(
      id: 'in-${DateTime.now().microsecondsSinceEpoch}',
      text: text,
      fromMe: false,
    ));
    chat.unread += 1;
    _typingUntil.remove(peerFingerprint); // a real message ends "typing"
    _moveToTop(chat);
    notifyListeners();

    if (!contact.muted) {
      final preview = (prefs?.notifPreview ?? true) ? text : 'New message';
      notifications?.push(title: chat.name, body: preview, chatId: chatId);
    }
  }

  void markRead(String chatId) {
    final chat = maybeChat(chatId);
    if (chat == null) return;
    if (chat.unread != 0) {
      chat.unread = 0;
      notifyListeners();
    }
    // Tell the peer we've read their messages (read receipt).
    final c = contactForChat(chat);
    if (c != null && c.code.isNotEmpty) onMarkRead?.call(c.code);
  }

  void _moveToTop(Chat chat) {
    _chats
      ..remove(chat)
      ..insert(0, chat);
  }
}
