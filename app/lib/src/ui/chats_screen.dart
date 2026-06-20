import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import '../services/chat_store.dart';
import 'conversation_screen.dart';
import 'settings_screen.dart';

/// The main screen: a list of conversations, backed by [ChatStore] so previews
/// always match the conversation contents.
class ChatsScreen extends StatelessWidget {
  const ChatsScreen({super.key, required this.core, required this.store});

  final StunnerCore core;
  final ChatStore store;

  Future<void> _newChat(BuildContext context) async {
    final controller = TextEditingController();
    final name = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New chat'),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: const InputDecoration(
            labelText: 'Contact name',
            hintText: 'e.g. Alice',
          ),
          onSubmitted: (v) => Navigator.pop(ctx, v),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, controller.text),
            child: const Text('Create'),
          ),
        ],
      ),
    );
    if (name == null || !context.mounted) return;
    final id = store.addChat(name);
    _open(context, id);
  }

  void _open(BuildContext context, String chatId) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => ConversationScreen(store: store, chatId: chatId),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Stunner'),
        actions: [
          IconButton(
            tooltip: 'Settings',
            icon: const Icon(Icons.settings_outlined),
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute<void>(builder: (_) => SettingsScreen(core: core)),
            ),
          ),
        ],
      ),
      body: ListenableBuilder(
        listenable: store,
        builder: (context, _) {
          final chats = store.chats;
          if (chats.isEmpty) {
            return const _EmptyState();
          }
          return ListView.separated(
            itemCount: chats.length,
            separatorBuilder: (_, __) => const Divider(height: 1),
            itemBuilder: (context, i) {
              final chat = chats[i];
              final last = chat.last;
              return ListTile(
                leading: CircleAvatar(
                  child: Text(chat.name.isEmpty ? '?' : chat.name[0].toUpperCase()),
                ),
                title: Text(chat.name),
                subtitle: Text(
                  last?.text ?? 'No messages yet',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    if (last != null)
                      Text(
                        _formatTime(last.time),
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    const SizedBox(height: 4),
                    if (chat.unread > 0) Badge(label: Text('${chat.unread}')),
                  ],
                ),
                onTap: () => _open(context, chat.id),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        tooltip: 'New chat',
        onPressed: () => _newChat(context),
        child: const Icon(Icons.add_comment_outlined),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.chat_bubble_outline,
              size: 64, color: Theme.of(context).disabledColor),
          const SizedBox(height: 12),
          const Text('No conversations yet'),
          const Text('Tap + to start one.'),
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
