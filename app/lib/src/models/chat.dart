// UI-side models. These mirror parts of the Go core's messaging/contact model
// (core/pkg/messaging, core/pkg/contact) and are driven by ChatStore.

/// Computes up-to-two-letter initials from a display name.
String initialsOf(String name) {
  final parts = name.trim().split(RegExp(r'\s+')).where((p) => p.isNotEmpty).toList();
  if (parts.isEmpty) return '?';
  if (parts.length == 1) return parts.first[0].toUpperCase();
  return (parts.first[0] + parts.last[0]).toUpperCase();
}

/// Delivery state of an outgoing message, mirroring messaging.DeliveryState.
enum DeliveryStatus { sending, sent, delivered, read, failed }

/// A person you can message. `code` is their `stunner:contact` URI (from a QR
/// code); `fingerprint` is derived from it for verification. The remaining
/// fields back the contact-profile screen.
class Contact {
  Contact({
    required this.id,
    required this.name,
    this.code = '',
    this.fingerprint = '',
    this.role = '',
    this.email = '',
    this.phone = '',
    this.online = false,
    this.muted = false,
    this.blocked = false,
  });

  final String id;
  String name;
  String code;
  final String fingerprint;
  String role;
  String email;
  String phone;
  bool online;
  bool muted;
  bool blocked;

  String get initials => initialsOf(name);
}

class Message {
  Message({
    required this.id,
    required this.text,
    required this.fromMe,
    DateTime? time,
    this.status = DeliveryStatus.sent,
    Map<String, int>? reactions,
  })  : time = time ?? DateTime.now(),
        reactions = reactions ?? {};

  final String id;
  final String text;
  final bool fromMe;
  final DateTime time;

  /// Delivery status (only meaningful for outgoing messages).
  DeliveryStatus status;

  /// Emoji -> count. A simple in-session reaction model.
  final Map<String, int> reactions;
}

class Chat {
  Chat({
    required this.id,
    required this.contactId,
    required this.name,
    List<Message>? messages,
    this.unread = 0,
  }) : messages = messages ?? [];

  final String id;
  final String contactId;
  String name;
  final List<Message> messages;
  int unread;

  Message? get last => messages.isEmpty ? null : messages.last;
}
