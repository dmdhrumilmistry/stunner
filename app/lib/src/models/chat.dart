// UI-side models. These mirror parts of the Go core's messaging/contact model
// (core/pkg/messaging, core/pkg/contact) and are driven by ChatStore.

/// Delivery state of an outgoing message, mirroring messaging.DeliveryState.
enum DeliveryStatus { sending, sent, delivered, read }

/// A person you can message. `code` is their `stunner:contact` URI (from a QR
/// code); `fingerprint` is derived from it for verification.
class Contact {
  Contact({
    required this.id,
    required this.name,
    this.code = '',
    this.fingerprint = '',
  });

  final String id;
  String name;
  final String code;
  final String fingerprint;
}

class Message {
  Message({
    required this.id,
    required this.text,
    required this.fromMe,
    DateTime? time,
    this.status = DeliveryStatus.sent,
  }) : time = time ?? DateTime.now();

  final String id;
  final String text;
  final bool fromMe;
  final DateTime time;

  /// Delivery status (only meaningful for outgoing messages).
  DeliveryStatus status;
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
