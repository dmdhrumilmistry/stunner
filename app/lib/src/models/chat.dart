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

  Map<String, dynamic> toMap() => {
        'id': id,
        'name': name,
        'code': code,
        'fingerprint': fingerprint,
        'role': role,
        'email': email,
        'phone': phone,
        'muted': muted,
        'blocked': blocked,
      };

  factory Contact.fromMap(Map<String, dynamic> m) => Contact(
        id: m['id'] as String? ?? '',
        name: m['name'] as String? ?? '',
        code: m['code'] as String? ?? '',
        fingerprint: m['fingerprint'] as String? ?? '',
        role: m['role'] as String? ?? '',
        email: m['email'] as String? ?? '',
        phone: m['phone'] as String? ?? '',
        muted: m['muted'] as bool? ?? false,
        blocked: m['blocked'] as bool? ?? false,
      ); // online is live presence; restored as offline
}

class Message {
  Message({
    required this.id,
    required this.text,
    required this.fromMe,
    DateTime? time,
    this.status = DeliveryStatus.sent,
    Map<String, int>? reactions,
    this.fileName,
    this.filePath,
  })  : time = time ?? DateTime.now(),
        reactions = reactions ?? {};

  final String id;
  final String text;
  final bool fromMe;
  final DateTime time;

  /// For a file message: the file's name and local path (null for text).
  final String? fileName;
  final String? filePath;

  bool get isFile => fileName != null;

  /// Delivery status (only meaningful for outgoing messages).
  DeliveryStatus status;

  /// Emoji -> count. A simple in-session reaction model.
  final Map<String, int> reactions;

  Map<String, dynamic> toMap() => {
        'id': id,
        'text': text,
        'fromMe': fromMe,
        'status': status.name,
        'timeMs': time.millisecondsSinceEpoch,
        if (fileName != null) 'fileName': fileName,
        if (filePath != null) 'filePath': filePath,
        if (reactions.isNotEmpty) 'reactions': reactions,
      };

  factory Message.fromMap(Map<String, dynamic> m) => Message(
        id: m['id'] as String? ?? '',
        text: m['text'] as String? ?? '',
        fromMe: m['fromMe'] as bool? ?? false,
        status: DeliveryStatus.values.asNameMap()[m['status']] ?? DeliveryStatus.sent,
        time: DateTime.fromMillisecondsSinceEpoch(m['timeMs'] as int? ?? 0),
        fileName: m['fileName'] as String?,
        filePath: m['filePath'] as String?,
        reactions: (m['reactions'] as Map?)
            ?.map((k, v) => MapEntry(k as String, (v as num).toInt())),
      );
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

  Map<String, dynamic> toMap() => {
        'id': id,
        'contactId': contactId,
        'name': name,
        'unread': unread,
        'messages': messages.map((m) => m.toMap()).toList(),
      };

  factory Chat.fromMap(Map<String, dynamic> m) => Chat(
        id: m['id'] as String? ?? '',
        contactId: m['contactId'] as String? ?? '',
        name: m['name'] as String? ?? '',
        unread: m['unread'] as int? ?? 0,
        messages: ((m['messages'] as List?) ?? const [])
            .map((e) => Message.fromMap((e as Map).cast<String, dynamic>()))
            .toList(),
      );
}
