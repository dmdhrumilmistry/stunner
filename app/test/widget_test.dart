import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:stunner/src/ffi/stunner_ffi.dart';
import 'package:stunner/src/models/chat.dart';
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

    expect(store.chats, isEmpty);
    expect(store.contacts, isEmpty);

    final carol = store.addContact(name: 'Carol', code: 'stunner:contact?k=abc', fingerprint: 'fp-carol');
    store.startChatWith(carol);

    await tester.pumpWidget(MaterialApp(
      home: HomeShell(core: core, store: store, appState: appState, notifications: notifications),
    ));
    await tester.pumpAndSettle();

    expect(find.text('Carol'), findsWidgets);
  });

  test('Onboarding gates the app and sets the profile', () {
    final appState = AppState();
    expect(appState.onboarded, isFalse);
    expect(appState.profile.name, isEmpty);

    appState.completeOnboarding(name: 'Jordan', username: '@jordan', contactCode: 'stunner:contact?k=xyz');
    expect(appState.onboarded, isTrue);
    expect(appState.profile.name, 'Jordan');
    expect(appState.profile.status, 'Available'); // defaulted when blank
    expect(appState.myContactCode, 'stunner:contact?k=xyz');
  });

  test('Sending without a runtime marks the message failed', () {
    final store = ChatStore();
    final c = store.addContact(name: 'Dana', code: 'stunner:contact?k=d', fingerprint: 'fp-d');
    final chatId = store.startChatWith(c);
    store.sendText(chatId, 'hello'); // no onSend wired
    expect(store.chatById(chatId).messages.single.status, DeliveryStatus.failed);
  });

  test('Outbound hook is invoked with the peer URI', () {
    final store = ChatStore();
    String? sentUri;
    String? sentMsgId;
    store.onSend = (uri, text, msgId) {
      sentUri = uri;
      sentMsgId = msgId;
    };
    final c = store.addContact(name: 'Eve', code: 'stunner:contact?k=e', fingerprint: 'fp-e');
    final chatId = store.startChatWith(c);
    store.sendText(chatId, 'hi');
    expect(sentUri, 'stunner:contact?k=e');
    final msg = store.chatById(chatId).messages.single;
    expect(msg.status, DeliveryStatus.sending);

    store.markSent(sentMsgId!);
    expect(store.chatById(chatId).messages.single.status, DeliveryStatus.sent);
  });

  test('Incoming message creates a chat and raises a notification', () {
    final notifications = NotificationService();
    final store = ChatStore(notifications: notifications);
    store.receiveFromPeer('fp-grace', 'stunner:contact?k=g', 'hi there');

    final contact = store.contactById('fp-grace');
    expect(contact, isNotNull);
    expect(contact!.code, 'stunner:contact?k=g'); // repliable
    expect(store.chats.single.messages.single.text, 'hi there');
    expect(store.chats.single.unread, 1);
    expect(notifications.unreadCount, 1);
  });

  test('Presence updates a contact', () {
    final store = ChatStore();
    store.addContact(name: 'Sam', code: 'x', fingerprint: 'fp-sam');
    expect(store.contactById('fp-sam')!.online, isFalse);
    store.setPresence('fp-sam', true);
    expect(store.contactById('fp-sam')!.online, isTrue);
  });

  test('Delete contact cascades to its chat', () {
    final store = ChatStore();
    final c = store.addContact(name: 'Kim', code: 'x', fingerprint: 'fp-kim');
    store.startChatWith(c);
    expect(store.chats, hasLength(1));
    store.deleteContact('fp-kim');
    expect(store.contacts, isEmpty);
    expect(store.chats, isEmpty);
  });
}
