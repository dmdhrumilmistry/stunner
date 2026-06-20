import 'package:flutter/material.dart';

import '../models/chat.dart';
import '../services/emoji.dart';

/// A single conversation view with a message composer.
///
/// The composer includes hooks for emoji (Unicode + animated) and file
/// attachment; sending is wired to the Go core's messaging service in a later
/// roadmap phase.
class ConversationScreen extends StatefulWidget {
  const ConversationScreen({super.key, required this.chat});

  final Chat chat;

  @override
  State<ConversationScreen> createState() => _ConversationScreenState();
}

class _ConversationScreenState extends State<ConversationScreen> {
  final _controller = TextEditingController();
  final _messages = <Message>[
    const Message(id: 'm1', text: 'Hey! 👋', fromMe: false),
    const Message(id: 'm2', text: 'Hi — end-to-end encrypted 🔒', fromMe: true),
  ];

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _send() {
    final text = expandShortcodes(_controller.text.trim());
    if (text.isEmpty) return;
    setState(() {
      _messages.add(Message(id: 'local-${_messages.length}', text: text, fromMe: true));
      _controller.clear();
    });
    // TODO: deliver via the core's node/link (core.sendText over FFI) once the
    // stateful runtime is exposed across the boundary.
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.chat.displayName)),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: _messages.length,
              itemBuilder: (context, i) => _Bubble(message: _messages[i]),
            ),
          ),
          SafeArea(child: _Composer(controller: _controller, onSend: _send)),
        ],
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.message});

  final Message message;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Align(
      alignment: message.fromMe ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: message.fromMe ? scheme.primaryContainer : scheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Text(message.text),
      ),
    );
  }
}

class _Composer extends StatelessWidget {
  const _Composer({required this.controller, required this.onSend});

  final TextEditingController controller;
  final VoidCallback onSend;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Row(
        children: [
          IconButton(
            tooltip: 'Emoji',
            icon: const Icon(Icons.emoji_emotions_outlined),
            onPressed: () {}, // TODO(phase 7): emoji_picker_flutter sheet
          ),
          IconButton(
            tooltip: 'Attach file',
            icon: const Icon(Icons.attach_file),
            onPressed: () {}, // TODO(phase 6): file_picker + filetransfer
          ),
          Expanded(
            child: TextField(
              controller: controller,
              minLines: 1,
              maxLines: 5,
              decoration: const InputDecoration(
                hintText: 'Message',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              onSubmitted: (_) => onSend(),
            ),
          ),
          IconButton(
            tooltip: 'Send',
            icon: const Icon(Icons.send),
            onPressed: onSend,
          ),
        ],
      ),
    );
  }
}
