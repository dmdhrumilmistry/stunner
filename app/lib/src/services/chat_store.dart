import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/chat.dart';

/// In-memory chat state shared by the chats list and the conversation view, so
/// the two always stay in sync. This is a local demo store: the live two-device
/// path (WebRTC over STUN/TURN via the Go core) is wired over FFI in a later
/// step. Outgoing messages simulate delivery/read receipts so the UX is
/// representative.
class ChatStore extends ChangeNotifier {
  ChatStore() {
    _seed();
  }

  final List<Chat> _chats = [];

  List<Chat> get chats => List.unmodifiable(_chats);

  Chat chatById(String id) => _chats.firstWhere((c) => c.id == id);

  void _seed() {
    final now = DateTime.now();
    _chats.addAll([
      Chat(
        id: 'c1',
        name: 'Alice',
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
        name: 'Bob',
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

  /// Adds a new conversation and returns its id.
  String addChat(String name) {
    final trimmed = name.trim();
    final id = 'c${DateTime.now().microsecondsSinceEpoch}';
    _chats.insert(0, Chat(id: id, name: trimmed.isEmpty ? 'New contact' : trimmed));
    notifyListeners();
    return id;
  }

  /// Sends a text message in a conversation and simulates delivery + read
  /// receipts (local demo).
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

  /// Marks a conversation as read (clears its unread badge).
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
