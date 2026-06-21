import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:stunner/src/core/chat_store.dart';
import 'package:stunner/src/core/event_pump.dart';
import 'package:stunner/src/ffi/stunner_ffi.dart';
import 'package:stunner/src/ui/chats_screen.dart';

void main() {
  testWidgets('Chats screen renders the app title', (tester) async {
    // Core is unavailable in tests (no native lib) — the app must still run.
    final core = StunnerCore.open();
    final store = ChatStore(core, PollingEventSource(core))..bootstrap();
    addTearDown(store.dispose);

    await tester.pumpWidget(
      MaterialApp(home: ChatsScreen(core: core, store: store)),
    );

    expect(find.text('Stunner'), findsOneWidget);
    expect(find.byIcon(Icons.settings_outlined), findsOneWidget);
  });
}
