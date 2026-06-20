import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:stunner/src/ffi/stunner_ffi.dart';
import 'package:stunner/src/services/chat_store.dart';
import 'package:stunner/src/ui/chats_screen.dart';

void main() {
  testWidgets('Chats screen renders seeded conversations', (tester) async {
    final core = StunnerCore.open();
    final store = ChatStore();
    await tester.pumpWidget(MaterialApp(home: ChatsScreen(core: core, store: store)));

    expect(find.text('Stunner'), findsOneWidget);
    expect(find.text('Alice'), findsOneWidget);
    expect(find.byIcon(Icons.add_comment_outlined), findsOneWidget);
  });

  test('Adding a contact and starting a chat', () {
    final store = ChatStore();
    final before = store.chats.length;
    final contact = store.addContact(name: 'Carol');
    expect(store.contacts.any((c) => c.name == 'Carol'), isTrue);

    final chatId = store.startChatWith(contact);
    expect(store.chats.length, before + 1);
    expect(store.chatById(chatId).name, 'Carol');

    // Starting again reuses the same chat.
    expect(store.startChatWith(contact), chatId);
    expect(store.chats.length, before + 1);
  });

  test('Delete chat and message', () {
    final store = ChatStore();
    final chat = store.chats.first;
    final msgId = chat.messages.first.id;
    store.deleteMessage(chat.id, msgId);
    expect(store.chatById(chat.id).messages.any((m) => m.id == msgId), isFalse);

    final count = store.chats.length;
    store.deleteChat(chat.id);
    expect(store.chats.length, count - 1);
  });
}
