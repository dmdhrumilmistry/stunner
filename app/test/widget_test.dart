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

  test('Delivery + read receipts update status (read does not regress)', () {
    final store = ChatStore();
    store.onSend = (uri, text, msgId) {}; // keep it "sending"
    final c = store.addContact(name: 'Ivy', code: 'stunner:contact?k=i', fingerprint: 'fp-ivy');
    final chatId = store.startChatWith(c);
    store.sendText(chatId, 'yo');
    final id = store.chatById(chatId).messages.single.id;

    store.markDelivered(id);
    expect(store.chatById(chatId).messages.single.status, DeliveryStatus.delivered);
    store.markReadByPeer('fp-ivy');
    expect(store.chatById(chatId).messages.single.status, DeliveryStatus.read);
    store.markDelivered(id); // must not regress read -> delivered
    expect(store.chatById(chatId).messages.single.status, DeliveryStatus.read);
  });

  test('Opening a chat triggers a read-receipt callback', () {
    final store = ChatStore();
    String? readUri;
    store.onMarkRead = (uri) => readUri = uri;
    final c = store.addContact(name: 'Jo', code: 'stunner:contact?k=j', fingerprint: 'fp-jo');
    final chatId = store.startChatWith(c);
    store.markRead(chatId);
    expect(readUri, 'stunner:contact?k=j');
  });

  test('Sending a file adds a file message and invokes the hook', () {
    final store = ChatStore();
    String? sentUri;
    String? sentPath;
    store.onSendFile = (uri, path, msgId) {
      sentUri = uri;
      sentPath = path;
    };
    final c = store.addContact(name: 'Liz', code: 'stunner:contact?k=l', fingerprint: 'fp-l');
    final chatId = store.startChatWith(c);
    store.sendFile(chatId, '/tmp/report.pdf');
    final m = store.chatById(chatId).messages.single;
    expect(m.isFile, isTrue);
    expect(m.fileName, 'report.pdf');
    expect(m.status, DeliveryStatus.sending);
    expect(sentUri, 'stunner:contact?k=l');
    expect(sentPath, '/tmp/report.pdf');
  });

  test('Typing: receive sets isTyping; an inbound message clears it', () {
    final store = ChatStore();
    store.receiveTyping('fp-t');
    expect(store.isTyping('fp-t'), isTrue);
    store.receiveFromPeer('fp-t', '', 'hey');
    expect(store.isTyping('fp-t'), isFalse);
  });

  test('Typing send respects the prefs toggle', () {
    final store = ChatStore();
    var called = 0;
    store.onTyping = (_) => called++;
    final c = store.addContact(name: 'T', code: 'stunner:contact?k=t', fingerprint: 'fp-t2');
    final chatId = store.startChatWith(c);

    store.prefs = Prefs()..typingIndicators = false;
    store.sendTyping(chatId);
    expect(called, 0);

    store.prefs = Prefs()..typingIndicators = true;
    store.sendTyping(chatId);
    expect(called, 1);
  });

  test('Notification preview is hidden when disabled', () {
    final notifications = NotificationService();
    final store = ChatStore(notifications: notifications);
    store.prefs = Prefs()..notifPreview = false;
    store.receiveFromPeer('fp-n', '', 'secret message');
    expect(notifications.items.first.body, 'New message');
  });

  test('Test connection invokes the diagnose hook and applies the result', () {
    final store = ChatStore();
    String? diagnosedUri;
    store.onDiagnose = (uri) => diagnosedUri = uri;
    final c = store.addContact(name: 'Ned', code: 'stunner:contact?k=n', fingerprint: 'fp-ned');
    final chatId = store.startChatWith(c);

    store.testConnection(chatId);
    expect(diagnosedUri, 'stunner:contact?k=n');
    expect(store.diagnosticFor('fp-ned')!.testing, isTrue);

    store.applyDiagnostic('fp-ned', false, 'Not found on the network yet.');
    final d = store.diagnosticFor('fp-ned')!;
    expect(d.testing, isFalse);
    expect(d.ok, isFalse);
    expect(d.message, contains('Not found'));

    store.clearDiagnostic('fp-ned');
    expect(store.diagnosticFor('fp-ned'), isNull);
  });

  test('Receiving a file creates an inbound file message', () {
    final store = ChatStore();
    store.receiveFileFromPeer('fp-m', 'stunner:contact?k=m', 'photo.jpg', '/data/photo.jpg');
    final m = store.chats.single.messages.single;
    expect(m.isFile, isTrue);
    expect(m.fileName, 'photo.jpg');
    expect(m.fromMe, isFalse);
  });

  test('App state + chat store serialize and restore (persistence)', () {
    final store = ChatStore();
    final c = store.addContact(name: 'Ada', code: 'stunner:contact?k=a', fingerprint: 'fp-a');
    final chatId = store.startChatWith(c);
    store.onSend = (u, t, m) {};
    store.sendText(chatId, 'hello');
    store.markSent(store.chatById(chatId).messages.single.id);

    final appState = AppState();
    appState.completeOnboarding(name: 'Ada', contactCode: 'stunner:contact?k=a');
    appState.setThemeMode(ThemeMode.dark);

    // Round-trip through the same JSON shape the runtime persists.
    final blob = {'app': appState.toMap(), 'store': store.toMap()};

    final store2 = ChatStore();
    final app2 = AppState();
    app2.restoreFromMap((blob['app'] as Map).cast<String, dynamic>());
    store2.restoreFromMap((blob['store'] as Map).cast<String, dynamic>());

    expect(app2.onboarded, isTrue);
    expect(app2.profile.name, 'Ada');
    expect(app2.themeMode, ThemeMode.dark);
    expect(app2.myContactCode, 'stunner:contact?k=a');
    expect(store2.contacts.single.name, 'Ada');
    final m = store2.chatById(chatId).messages.single;
    expect(m.text, 'hello');
    expect(m.status, DeliveryStatus.sent);
  });
}
