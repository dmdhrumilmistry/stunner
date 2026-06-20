import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:stunner/src/ffi/stunner_ffi.dart';
import 'package:stunner/src/services/app_state.dart';
import 'package:stunner/src/services/chat_store.dart';
import 'package:stunner/src/services/notification_service.dart';
import 'package:stunner/src/ui/home_shell.dart';

void main() {
  testWidgets('Home shell starts empty and shows added chats', (tester) async {
    final core = StunnerCore.open();
    final store = ChatStore();
    final appState = AppState();
    final notifications = NotificationService();

    // Empty by default (no seeded Alice/Bob).
    expect(store.chats, isEmpty);
    expect(store.contacts, isEmpty);

    final carol = store.addContact(name: 'Carol');
    store.startChatWith(carol);

    await tester.pumpWidget(MaterialApp(
      home: HomeShell(core: core, store: store, appState: appState, notifications: notifications),
    ));
    await tester.pumpAndSettle();

    expect(find.text('Carol'), findsWidgets);
    expect(find.byIcon(Icons.edit_outlined), findsWidgets);
  });

  test('Adding a contact and starting a chat', () {
    final store = ChatStore();
    final contact = store.addContact(name: 'Carol');
    expect(store.contacts.any((c) => c.name == 'Carol'), isTrue);

    final chatId = store.startChatWith(contact);
    expect(store.chats.length, 1);
    expect(store.chatById(chatId).name, 'Carol');

    // Starting again reuses the same chat.
    expect(store.startChatWith(contact), chatId);
    expect(store.chats.length, 1);
  });

  test('Delete chat, message, and contact (cascades)', () {
    final store = ChatStore();
    final contact = store.addContact(name: 'Dana');
    final chatId = store.startChatWith(contact);
    store.receiveText(chatId, 'hello');
    final msgId = store.chatById(chatId).messages.first.id;

    store.deleteMessage(chatId, msgId);
    expect(store.chatById(chatId).messages.any((m) => m.id == msgId), isFalse);

    store.deleteContact(contact.id);
    expect(store.contacts, isEmpty);
    expect(store.chats, isEmpty); // conversation removed with the contact
  });

  test('Reactions toggle on and off', () {
    final store = ChatStore();
    final contact = store.addContact(name: 'Eve');
    final chatId = store.startChatWith(contact);
    store.receiveText(chatId, 'react to me');
    final msgId = store.chatById(chatId).messages.first.id;

    store.toggleReaction(chatId, msgId, '👍');
    expect(store.chatById(chatId).messages.first.reactions['👍'], 1);
    store.toggleReaction(chatId, msgId, '👍');
    expect(store.chatById(chatId).messages.first.reactions.containsKey('👍'), isFalse);
  });

  test('Mute and block toggle on a contact', () {
    final store = ChatStore();
    final id = store.addContact(name: 'Frank').id;
    expect(store.contactById(id)!.muted, isFalse);
    store.toggleMute(id);
    expect(store.contactById(id)!.muted, isTrue);
    store.toggleBlock(id);
    expect(store.contactById(id)!.blocked, isTrue);
  });

  test('Incoming message raises a notification', () {
    final notifications = NotificationService();
    final store = ChatStore(notifications: notifications);
    final contact = store.addContact(name: 'Grace');
    final chatId = store.startChatWith(contact);

    store.receiveText(chatId, 'hi there');
    expect(store.chatById(chatId).unread, 1);
    expect(notifications.unreadCount, 1);
    expect(notifications.items.first.body, 'hi there');
  });

  test('Profile edit updates app state', () {
    final appState = AppState();
    appState.updateProfile(
      name: 'New Name',
      username: '@newname',
      status: 'Busy',
      email: 'new@stunner.app',
    );
    expect(appState.profile.name, 'New Name');
    expect(appState.profile.username, '@newname');
    expect(appState.profile.status, 'Busy');
    expect(appState.profile.initials, 'NN');
  });
}
