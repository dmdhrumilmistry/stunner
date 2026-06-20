import 'package:flutter/material.dart';

import 'src/ffi/stunner_ffi.dart';
import 'src/ui/chats_screen.dart';

void main() {
  // Open the native Stunner core (FFI smoke test). Degrades gracefully if the
  // library has not been built yet.
  final core = StunnerCore.open();
  // ignore: avoid_print
  print('Stunner core: ${core.version()}');

  runApp(StunnerApp(core: core));
}

class StunnerApp extends StatelessWidget {
  const StunnerApp({super.key, required this.core});

  final StunnerCore core;

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
      home: ChatsScreen(core: core),
    );
  }
}
