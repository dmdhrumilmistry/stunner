// UI-side models. These mirror parts of the Go core's messaging model
// (core/pkg/messaging) and will eventually be populated from the core over FFI.

class Chat {
  const Chat({
    required this.id,
    required this.displayName,
    required this.lastMessage,
    this.unread = 0,
  });

  final String id;
  final String displayName;
  final String lastMessage;
  final int unread;
}

class Message {
  const Message({
    required this.id,
    required this.text,
    required this.fromMe,
  });

  final String id;
  final String text;
  final bool fromMe;
}

/// Placeholder data so the UI shell is browsable before the core is wired up.
const sampleChats = <Chat>[
  Chat(
    id: 'c1',
    displayName: 'Alice',
    lastMessage: 'See you tomorrow 👋',
    unread: 2,
  ),
  Chat(
    id: 'c2',
    displayName: 'Bob',
    lastMessage: 'Sent you a file 📎',
  ),
];
