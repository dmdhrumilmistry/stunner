import 'package:flutter/material.dart';

import '../core/chat_store.dart';
import '../models/chat.dart';

/// A single conversation view with a message composer. Messages and sending are
/// backed by the [ChatStore] (Go core over FFI).
class ConversationScreen extends StatefulWidget {
  const ConversationScreen({super.key, required this.convId, required this.store});

  final String convId;
  final ChatStore store;

  @override
  State<ConversationScreen> createState() => _ConversationScreenState();
}

class _ConversationScreenState extends State<ConversationScreen> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _send() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    widget.store.send(widget.convId, text);
    _controller.clear();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: ListenableBuilder(
          listenable: widget.store,
          builder: (context, _) {
            final chat = widget.store.chatFor(widget.convId);
            return Text(chat?.displayName ?? 'Conversation');
          },
        ),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListenableBuilder(
              listenable: widget.store,
              builder: (context, _) {
                final messages = widget.store.messagesFor(widget.convId);
                return ListView.builder(
                  padding: const EdgeInsets.all(12),
                  itemCount: messages.length,
                  itemBuilder: (context, i) => _Bubble(message: messages[i]),
                );
              },
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
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Flexible(child: Text(message.text)),
            if (message.fromMe) ...[
              const SizedBox(width: 6),
              Icon(_stateIcon(message.state), size: 14, color: scheme.outline),
            ],
          ],
        ),
      ),
    );
  }

  IconData _stateIcon(String state) {
    switch (state) {
      case 'FAILED':
        return Icons.error_outline;
      case 'QUEUED':
        return Icons.schedule;
      case 'DELIVERED':
      case 'READ':
        return Icons.done_all;
      default:
        return Icons.check;
    }
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
