import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/chat.dart';
import 'notification_service.dart';

/// In-memory chats + contacts shared by every screen, so the list and the
/// conversation always agree. Local demo store: the live two-device path
/// (WebRTC over STUN/TURN via the Go core) is wired over FFI in a later step.
///
/// The app starts empty — add a contact to begin. Incoming messages flow through
/// [receiveText], which raises an in-app notification via [notifications].
class ChatStore extends ChangeNotifier {
  ChatStore({this.notifications});

  /// Optional sink for in-app live notifications on incoming messages.
  final NotificationService? notifications;

  final List<Contact> _contacts = [];
  final List<Chat> _chats = [];

  // A few canned peer replies so the demo can exercise the live-notification
  // path until the GUI is wired to the real runtime over FFI.
  static const _demoReplies = [
    'Got it 👍',
    'Sounds good!',
    'On it 🔒',
    'Thanks for the update',
    'Let me check and get back to you',
  ];
  int _replyCursor = 0;

  List<Contact> get contacts => List.unmodifiable(_contacts);
  List<Chat> get chats => List.unmodifiable(_chats);

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
  /// when present, otherwise by name. New contacts are marked online so the demo
  /// can exercise live delivery; real presence arrives with the FFI runtime.
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
      existing.first.name = cleanName;
      notifyListeners();
      return existing.first;
    }
    final contact = Contact(
      id: fingerprint.isNotEmpty ? fingerprint : 'k-${DateTime.now().microsecondsSinceEpoch}',
      name: cleanName,
      code: code,
      fingerprint: fingerprint,
      role: role,
      online: true,
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

  /// Sends a text and simulates delivery + read receipts (local demo). If the
  /// peer is online, a demo reply arrives shortly after to exercise the
  /// live-notification path.
  void sendText(String chatId, String text) {
    final body = text.trim();
    if (body.isEmpty) return;
    final chat = chatById(chatId);
    final msg = Message(
      id: 'local-${DateTime.now().microsecondsSinceEpoch}',
      text: body,
      fromMe: true,
      status: DeliveryStatus.sending,
    );
    chat.messages.add(msg);
    _moveToTop(chat);
    notifyListeners();

    _advance(msg, DeliveryStatus.sent, 150);
    _advance(msg, DeliveryStatus.delivered, 700);
    _advance(msg, DeliveryStatus.read, 1600);

    final contact = contactForChat(chat);
    if (contact != null && contact.online && !contact.blocked) {
      final reply = _demoReplies[_replyCursor++ % _demoReplies.length];
      Future.delayed(const Duration(milliseconds: 2400), () => receiveText(chatId, reply));
    }
  }

  /// Delivers an incoming message into a conversation, bumping its unread count
  /// and raising an in-app notification. This is the hook the live FFI runtime
  /// will call for real peer messages.
  void receiveText(String chatId, String text) {
    final chat = maybeChat(chatId);
    if (chat == null) return;
    chat.messages.add(Message(
      id: 'in-${DateTime.now().microsecondsSinceEpoch}',
      text: text,
      fromMe: false,
    ));
    chat.unread += 1;
    _moveToTop(chat);
    notifyListeners();

    final contact = contactForChat(chat);
    if (contact == null || !contact.muted) {
      notifications?.push(title: chat.name, body: text, chatId: chatId);
    }
  }

  void _advance(Message m, DeliveryStatus to, int ms) {
    Future.delayed(Duration(milliseconds: ms), () {
      if (m.status.index < to.index) {
        m.status = to;
        notifyListeners();
      }
    });
  }

  void markRead(String chatId) {
    final chat = maybeChat(chatId);
    if (chat != null && chat.unread != 0) {
      chat.unread = 0;
      notifyListeners();
    }
  }

  void _moveToTop(Chat chat) {
    _chats
      ..remove(chat)
      ..insert(0, chat);
  }
}
