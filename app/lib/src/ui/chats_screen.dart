import 'package:flutter/material.dart';

import '../core/chat_store.dart';
import '../ffi/stunner_ffi.dart';
import 'conversation_screen.dart';
import 'settings_screen.dart';

/// The main screen: a list of conversations, backed by the [ChatStore].
class ChatsScreen extends StatelessWidget {
  const ChatsScreen({super.key, required this.core, required this.store});

  final StunnerCore core;
  final ChatStore store;

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
              MaterialPageRoute<void>(
                builder: (_) => SettingsScreen(core: core),
              ),
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
              return ListTile(
                leading: CircleAvatar(
                  child: Text(chat.displayName.isEmpty
                      ? '?'
                      : chat.displayName[0].toUpperCase()),
                ),
                title: Row(
                  children: [
                    Flexible(child: Text(chat.displayName, overflow: TextOverflow.ellipsis)),
                    if (chat.connected)
                      const Padding(
                        padding: EdgeInsets.only(left: 6),
                        child: Icon(Icons.circle, size: 8, color: Colors.green),
                      ),
                  ],
                ),
                subtitle: Text(
                  chat.lastMessage,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: chat.unread > 0 ? Badge(label: Text('${chat.unread}')) : null,
                onTap: () {
                  store.markRead(chat.convId);
                  Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) => ConversationScreen(convId: chat.convId, store: store),
                    ),
                  );
                },
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

  Future<void> _newChat(BuildContext context) async {
    final controller = TextEditingController();
    final uri = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New chat'),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: const InputDecoration(
            labelText: 'Peer contact URI (stunner:contact?...)',
            hintText: 'Paste the code from their My identity screen',
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, controller.text.trim()),
            child: const Text('Connect'),
          ),
        ],
      ),
    );
    if (uri == null || uri.isEmpty || !context.mounted) return;
    try {
      final convId = store.connect(uri);
      if (!context.mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => ConversationScreen(convId: convId, store: store),
        ),
      );
    } on Object catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not connect: $e')),
      );
    }
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.forum_outlined,
                size: 48, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 12),
            const Text(
              'No conversations yet.\nTap + to connect to a peer by their contact code.',
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}
