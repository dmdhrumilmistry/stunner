import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import '../models/chat.dart';
import '../services/chat_store.dart';
import 'conversation_screen.dart';
import 'settings_screen.dart';

/// Main screen: the conversation list, backed by [ChatStore]. Chats are always
/// tied to a contact (you start a chat by picking/adding a contact, not "no
/// one"). Swipe a row to delete it.
class ChatsScreen extends StatelessWidget {
  const ChatsScreen({super.key, required this.core, required this.store});

  final StunnerCore core;
  final ChatStore store;

  void _open(BuildContext context, String chatId) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => ConversationScreen(store: store, chatId: chatId),
      ),
    );
  }

  /// Pick an existing contact to chat with, or add a new one.
  Future<void> _startChat(BuildContext context) async {
    const addSentinel = '__add__';
    final choice = await showModalBottomSheet<String>(
      context: context,
      builder: (ctx) {
        final contacts = store.contacts;
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.person_add_alt),
                title: const Text('Add new contact'),
                onTap: () => Navigator.pop(ctx, addSentinel),
              ),
              const Divider(height: 1),
              if (contacts.isEmpty)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Text('No contacts yet — add one to start chatting.'),
                ),
              for (final c in contacts)
                ListTile(
                  leading: CircleAvatar(child: Text(_initial(c.name))),
                  title: Text(c.name),
                  subtitle: c.fingerprint.isEmpty
                      ? null
                      : Text(c.fingerprint, maxLines: 1, overflow: TextOverflow.ellipsis),
                  onTap: () => Navigator.pop(ctx, c.id),
                ),
            ],
          ),
        );
      },
    );

    if (choice == null || !context.mounted) return;

    if (choice == addSentinel) {
      final contact = await _addContact(context);
      if (contact != null && context.mounted) {
        _open(context, store.startChatWith(contact));
      }
      return;
    }

    final contact = store.contacts.where((c) => c.id == choice);
    if (contact.isNotEmpty) {
      _open(context, store.startChatWith(contact.first));
    }
  }

  Future<Contact?> _addContact(BuildContext context) async {
    final nameCtl = TextEditingController();
    final codeCtl = TextEditingController();
    String? error;

    return showDialog<Contact>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('Add contact'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: nameCtl,
                autofocus: true,
                decoration: const InputDecoration(labelText: 'Name'),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: codeCtl,
                decoration: const InputDecoration(
                  labelText: 'Contact code (optional)',
                  hintText: 'stunner:contact?...',
                ),
              ),
              if (error != null) ...[
                const SizedBox(height: 8),
                Text(error!, style: TextStyle(color: Theme.of(ctx).colorScheme.error)),
              ],
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
            FilledButton(
              onPressed: () {
                final code = codeCtl.text.trim();
                var fingerprint = '';
                var name = nameCtl.text.trim();
                if (code.isNotEmpty) {
                  try {
                    final info = core.validateContactURI(code);
                    fingerprint = info.fingerprint;
                    if (name.isEmpty) name = info.handle;
                  } on FormatException catch (e) {
                    setState(() => error = 'Invalid contact code: ${e.message}');
                    return;
                  }
                }
                if (name.isEmpty) {
                  setState(() => error = 'Enter a name or a contact code.');
                  return;
                }
                Navigator.pop(
                  ctx,
                  store.addContact(name: name, code: code, fingerprint: fingerprint),
                );
              },
              child: const Text('Add'),
            ),
          ],
        ),
      ),
    );
  }

  Future<bool> _confirmDelete(BuildContext context, String name) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Delete chat with $name?'),
        content: const Text('This removes the conversation from this device.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Delete')),
        ],
      ),
    );
    return ok ?? false;
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
              return Dismissible(
                key: ValueKey(chat.id),
                direction: DismissDirection.endToStart,
                background: Container(
                  color: Theme.of(context).colorScheme.errorContainer,
                  alignment: Alignment.centerRight,
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  child: const Icon(Icons.delete_outline),
                ),
                confirmDismiss: (_) => _confirmDelete(context, chat.name),
                onDismissed: (_) => store.deleteChat(chat.id),
                child: ListTile(
                  leading: CircleAvatar(child: Text(_initial(chat.name))),
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
                        Text(_formatTime(last.time),
                            style: Theme.of(context).textTheme.bodySmall),
                      const SizedBox(height: 4),
                      if (chat.unread > 0) Badge(label: Text('${chat.unread}')),
                    ],
                  ),
                  onTap: () => _open(context, chat.id),
                ),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        tooltip: 'New chat',
        onPressed: () => _startChat(context),
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
          Icon(Icons.chat_bubble_outline, size: 64, color: Theme.of(context).disabledColor),
          const SizedBox(height: 12),
          const Text('No conversations yet'),
          const Text('Tap + to add a contact and start chatting.'),
        ],
      ),
    );
  }
}

String _initial(String s) => s.isEmpty ? '?' : s[0].toUpperCase();

String _formatTime(DateTime t) {
  final h = t.hour.toString().padLeft(2, '0');
  final m = t.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
