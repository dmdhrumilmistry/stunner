import 'package:flutter/material.dart';

import '../models/chat.dart';
import '../services/chat_store.dart';
import '../services/emoji.dart';

/// A single conversation: messages from [ChatStore], a working composer (emoji
/// picker, attach, send), read-receipt ticks, and long-press to delete a
/// message.
class ConversationScreen extends StatefulWidget {
  const ConversationScreen({super.key, required this.store, required this.chatId});

  final ChatStore store;
  final String chatId;

  @override
  State<ConversationScreen> createState() => _ConversationScreenState();
}

class _ConversationScreenState extends State<ConversationScreen> {
  final _controller = TextEditingController();
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      widget.store.markRead(widget.chatId);
    });
  }

  @override
  void dispose() {
    _controller.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _send() {
    final text = expandShortcodes(_controller.text);
    if (text.trim().isEmpty) return;
    widget.store.sendText(widget.chatId, text);
    _controller.clear();
    WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToBottom());
  }

  void _scrollToBottom() {
    if (_scroll.hasClients) {
      _scroll.animateTo(
        _scroll.position.maxScrollExtent,
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
      );
    }
  }

  Future<void> _deleteMessage(Message m) async {
    final ok = await showModalBottomSheet<bool>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Wrap(
          children: [
            ListTile(
              leading: const Icon(Icons.delete_outline),
              title: const Text('Delete message'),
              onTap: () => Navigator.pop(ctx, true),
            ),
            ListTile(
              leading: const Icon(Icons.close),
              title: const Text('Cancel'),
              onTap: () => Navigator.pop(ctx, false),
            ),
          ],
        ),
      ),
    );
    if (ok == true) {
      widget.store.deleteMessage(widget.chatId, m.id);
    }
  }

  Future<void> _pickEmoji() async {
    final picked = await showModalBottomSheet<String>(
      context: context,
      builder: (ctx) => SafeArea(
        child: GridView.count(
          crossAxisCount: 8,
          shrinkWrap: true,
          padding: const EdgeInsets.all(8),
          children: [
            for (final e in pickerEmojis)
              IconButton(
                onPressed: () => Navigator.pop(ctx, e),
                icon: Text(e, style: const TextStyle(fontSize: 24)),
              ),
          ],
        ),
      ),
    );
    if (picked == null) return;
    final sel = _controller.selection;
    final text = _controller.text;
    if (sel.isValid) {
      _controller.text = text.replaceRange(sel.start, sel.end, picked);
      _controller.selection = TextSelection.collapsed(offset: sel.start + picked.length);
    } else {
      _controller.text = text + picked;
      _controller.selection = TextSelection.collapsed(offset: _controller.text.length);
    }
  }

  void _attach() {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('File sharing needs a live connection (coming soon).')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: ListenableBuilder(
          listenable: widget.store,
          builder: (_, __) => Text(widget.store.chatById(widget.chatId).name),
        ),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListenableBuilder(
              listenable: widget.store,
              builder: (context, _) {
                final messages = widget.store.chatById(widget.chatId).messages;
                if (messages.isEmpty) {
                  return const Center(child: Text('Say hello 👋'));
                }
                return ListView.builder(
                  controller: _scroll,
                  padding: const EdgeInsets.all(12),
                  itemCount: messages.length,
                  itemBuilder: (context, i) {
                    final m = messages[i];
                    return GestureDetector(
                      onLongPress: () => _deleteMessage(m),
                      child: _Bubble(message: m),
                    );
                  },
                );
              },
            ),
          ),
          SafeArea(
            child: _Composer(
              controller: _controller,
              onSend: _send,
              onEmoji: _pickEmoji,
              onAttach: _attach,
            ),
          ),
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
        constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.78),
        decoration: BoxDecoration(
          color: message.fromMe ? scheme.primaryContainer : scheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Text(message.text),
            const SizedBox(height: 2),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(_formatTime(message.time), style: Theme.of(context).textTheme.bodySmall),
                if (message.fromMe) ...[
                  const SizedBox(width: 4),
                  _ReceiptTick(status: message.status),
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ReceiptTick extends StatelessWidget {
  const _ReceiptTick({required this.status});

  final DeliveryStatus status;

  @override
  Widget build(BuildContext context) {
    switch (status) {
      case DeliveryStatus.sending:
        return const Icon(Icons.schedule, size: 14, color: Colors.grey);
      case DeliveryStatus.sent:
        return const Icon(Icons.check, size: 14, color: Colors.grey);
      case DeliveryStatus.delivered:
        return const Icon(Icons.done_all, size: 14, color: Colors.grey);
      case DeliveryStatus.read:
        return Icon(Icons.done_all, size: 14, color: Colors.blue.shade400);
    }
  }
}

class _Composer extends StatelessWidget {
  const _Composer({
    required this.controller,
    required this.onSend,
    required this.onEmoji,
    required this.onAttach,
  });

  final TextEditingController controller;
  final VoidCallback onSend;
  final VoidCallback onEmoji;
  final VoidCallback onAttach;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Row(
        children: [
          IconButton(
            tooltip: 'Emoji',
            icon: const Icon(Icons.emoji_emotions_outlined),
            onPressed: onEmoji,
          ),
          IconButton(
            tooltip: 'Attach file',
            icon: const Icon(Icons.attach_file),
            onPressed: onAttach,
          ),
          Expanded(
            child: TextField(
              controller: controller,
              minLines: 1,
              maxLines: 5,
              textInputAction: TextInputAction.send,
              decoration: const InputDecoration(
                hintText: 'Message',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              onSubmitted: (_) => onSend(),
            ),
          ),
          IconButton(tooltip: 'Send', icon: const Icon(Icons.send), onPressed: onSend),
        ],
      ),
    );
  }
}

String _formatTime(DateTime t) {
  final h = t.hour.toString().padLeft(2, '0');
  final m = t.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
