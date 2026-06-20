import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/chat.dart';

/// In-memory chats + contacts shared by every screen, so the list and the
/// conversation always agree. Local demo store: the live two-device path
/// (WebRTC over STUN/TURN via the Go core) is wired over FFI in a later step.
class ChatStore extends ChangeNotifier {
  ChatStore() {
    _seed();
  }

  final List<Contact> _contacts = [];
  final List<Chat> _chats = [];

  List<Contact> get contacts => List.unmodifiable(_contacts);
  List<Chat> get chats => List.unmodifiable(_chats);

  Chat chatById(String id) => _chats.firstWhere((c) => c.id == id);

  void _seed() {
    final now = DateTime.now();
    final alice = Contact(id: 'k-alice', name: 'Alice');
    final bob = Contact(id: 'k-bob', name: 'Bob');
    _contacts.addAll([alice, bob]);
    _chats.addAll([
      Chat(
        id: 'c1',
        contactId: alice.id,
        name: alice.name,
        messages: [
          Message(
            id: 'm1',
            text: 'Hey! 👋',
            fromMe: false,
            time: now.subtract(const Duration(minutes: 6)),
          ),
          Message(
            id: 'm2',
            text: 'Hi — end-to-end encrypted 🔒',
            fromMe: true,
            time: now.subtract(const Duration(minutes: 5)),
            status: DeliveryStatus.read,
          ),
        ],
      ),
      Chat(
        id: 'c2',
        contactId: bob.id,
        name: bob.name,
        unread: 1,
        messages: [
          Message(
            id: 'm3',
            text: 'Sent you the notes 📝',
            fromMe: false,
            time: now.subtract(const Duration(hours: 2)),
          ),
        ],
      ),
    ]);
  }

  // --- contacts ---

  /// Adds (or updates) a contact and returns it. Deduplicated by fingerprint
  /// when present, otherwise by name.
  Contact addContact({required String name, String code = '', String fingerprint = ''}) {
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
    );
    _contacts.add(contact);
    notifyListeners();
    return contact;
  }

  void deleteContact(String contactId) {
    _contacts.removeWhere((c) => c.id == contactId);
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
    chatById(chatId).messages.removeWhere((m) => m.id == messageId);
    notifyListeners();
  }

  /// Sends a text and simulates delivery + read receipts (local demo).
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
    final chat = chatById(chatId);
    if (chat.unread != 0) {
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
