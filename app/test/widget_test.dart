import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:stunner/src/ffi/stunner_ffi.dart';
import 'package:stunner/src/services/chat_store.dart';
import 'package:stunner/src/ui/chats_screen.dart';

void main() {
  testWidgets('Chats screen renders seeded conversations', (tester) async {
    // Core is unavailable in tests (no native lib) — the app must still run.
    final core = StunnerCore.open();
    final store = ChatStore();
    await tester.pumpWidget(MaterialApp(home: ChatsScreen(core: core, store: store)));

    expect(find.text('Stunner'), findsOneWidget);
    expect(find.text('Alice'), findsOneWidget);
    expect(find.byIcon(Icons.add_comment_outlined), findsOneWidget);
  });

  testWidgets('New chat can be created', (tester) async {
    final core = StunnerCore.open();
    final store = ChatStore();
    await tester.pumpWidget(MaterialApp(home: ChatsScreen(core: core, store: store)));

    expect(store.chats.length, 2);
    store.addChat('Carol');
    expect(store.chats.first.name, 'Carol');
  });
}
