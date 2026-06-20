import 'package:flutter/foundation.dart';

/// A single in-app notification (an incoming message, mostly).
class AppNotification {
  AppNotification({
    required this.id,
    required this.title,
    required this.body,
    required this.chatId,
    DateTime? time,
  }) : time = time ?? DateTime.now();

  final String id;
  final String title;
  final String body;
  final String chatId;
  final DateTime time;
  bool read = false;
}

/// In-app live notifications. When a message arrives the store calls [push];
/// the UI listens and surfaces a banner plus an unread badge / notification
/// center. This is the app-side plumbing the live FFI path will feed once the
/// runtime is wired to the GUI; today it is driven by the demo store.
class NotificationService extends ChangeNotifier {
  final List<AppNotification> _items = [];

  /// Monotonic counter bumped on every [push], so the UI can detect a *new*
  /// notification (to show a one-shot banner) without diffing the list.
  int version = 0;
  AppNotification? latest;

  List<AppNotification> get items => List.unmodifiable(_items.reversed);
  int get unreadCount => _items.where((n) => !n.read).length;

  void push({required String title, required String body, required String chatId}) {
    final n = AppNotification(
      id: 'n-${DateTime.now().microsecondsSinceEpoch}',
      title: title,
      body: body,
      chatId: chatId,
    );
    _items.add(n);
    latest = n;
    version++;
    notifyListeners();
  }

  void markRead(String id) {
    for (final n in _items) {
      if (n.id == id) n.read = true;
    }
    notifyListeners();
  }

  void markAllRead() {
    for (final n in _items) {
      n.read = true;
    }
    notifyListeners();
  }

  void clear() {
    _items.clear();
    latest = null;
    notifyListeners();
  }
}
