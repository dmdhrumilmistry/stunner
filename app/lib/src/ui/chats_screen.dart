import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import '../models/chat.dart';
import 'conversation_screen.dart';
import 'settings_screen.dart';

/// The main screen: a list of conversations.
class ChatsScreen extends StatelessWidget {
  const ChatsScreen({super.key, required this.core});

  final StunnerCore core;

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
      body: ListView.separated(
        itemCount: sampleChats.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, i) {
          final chat = sampleChats[i];
          return ListTile(
            leading: CircleAvatar(child: Text(chat.displayName[0])),
            title: Text(chat.displayName),
            subtitle: Text(
              chat.lastMessage,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: chat.unread > 0
                ? Badge(label: Text('${chat.unread}'))
                : null,
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => ConversationScreen(chat: chat),
              ),
            ),
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        tooltip: 'New chat',
        onPressed: () {},
        child: const Icon(Icons.add_comment_outlined),
      ),
    );
  }
}
