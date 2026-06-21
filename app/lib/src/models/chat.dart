// UI-side models, populated from the Go core over FFI (see core/pkg/session and
// core/pkg/messaging).

class Chat {
  Chat({
    required this.convId,
    required this.displayName,
    this.peerFingerprint = '',
    this.peerContactURI = '',
    this.isContact = false,
    this.lastMessage = '',
    this.unread = 0,
    this.connected = false,
  });

  /// Conversation id (stable key). For 1:1 chats this is the peer fingerprint.
  final String convId;
  String displayName;

  /// The peer's identity fingerprint; empty until a message/connection arrives.
  String peerFingerprint;

  /// The peer's `stunner:contact` URI when known (e.g. from the connect flow).
  String peerContactURI;

  /// Whether the peer is in the user's contacts. Messages from non-contacts
  /// still create a chat (with this false).
  bool isContact;

  String lastMessage;
  int unread;
  bool connected;

  /// Builds a chat for an inbound message from a peer not yet known locally.
  factory Chat.unknownFrom(String convId, String peerFingerprint) {
    final short = peerFingerprint.replaceAll(' ', '');
    final label = short.length >= 6 ? short.substring(0, 6) : short;
    return Chat(
      convId: convId,
      displayName: 'Unknown · $label',
      peerFingerprint: peerFingerprint,
      isContact: false,
    );
  }
}

class Message {
  Message({
    required this.id,
    required this.text,
    required this.fromMe,
    this.state = 'SENT',
  });

  final String id;
  final String text;
  final bool fromMe;

  /// Delivery state for outgoing messages: QUEUED | SENT | DELIVERED | FAILED.
  String state;
}
