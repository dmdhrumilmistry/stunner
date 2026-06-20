import 'package:flutter/material.dart';

import 'src/ffi/stunner_ffi.dart';
import 'src/services/chat_store.dart';
import 'src/ui/chats_screen.dart';

void main() {
  // Open the native Stunner core (FFI). Degrades gracefully if the library has
  // not been built yet.
  final core = StunnerCore.open();
  // ignore: avoid_print
  print('Stunner core: ${core.version()}');

  runApp(StunnerApp(core: core, store: ChatStore()));
}

class StunnerApp extends StatelessWidget {
  const StunnerApp({super.key, required this.core, required this.store});

  final StunnerCore core;
  final ChatStore store;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Stunner',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF4C5FD5),
        useMaterial3: true,
        brightness: Brightness.light,
      ),
      darkTheme: ThemeData(
        colorSchemeSeed: const Color(0xFF4C5FD5),
        useMaterial3: true,
        brightness: Brightness.dark,
      ),
      home: ChatsScreen(core: core, store: store),
    );
  }
}
