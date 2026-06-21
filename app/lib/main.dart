import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider/path_provider.dart';

import 'src/core/chat_store.dart';
import 'src/core/event_pump.dart';
import 'src/ffi/stunner_ffi.dart';
import 'src/ui/chats_screen.dart';

const _vaultKeyName = 'stunner_vault_key';
const _iceServersName = 'stunner_ice_servers';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Open the native Stunner core. Degrades gracefully if the library has not
  // been built yet (the UI still runs, just without networking).
  final core = StunnerCore.open();
  // ignore: avoid_print
  print('Stunner core: ${core.version()}');

  final store = await _bootCore(core);

  runApp(StunnerApp(core: core, store: store));
}

/// Starts the runtime (account dir + secure key + ICE settings) and wires the
/// event source into a [ChatStore]. Returns a store even in degraded mode so the
/// UI shell remains usable.
Future<ChatStore> _bootCore(StunnerCore core) async {
  final source = PollingEventSource(core);
  final store = ChatStore(core, source);

  if (core.available) {
    try {
      final dir = await getApplicationSupportDirectory();
      final keyHex = await _loadOrCreateKey();
      const secure = FlutterSecureStorage();
      final ice = await secure.read(key: _iceServersName) ?? '';
      core.start(accountDir: dir.path, keyHex: keyHex, iceServersJson: ice);
    } on Object catch (e) {
      // ignore: avoid_print
      print('Stunner core start failed: $e');
    }
  }

  store.bootstrap();
  return store;
}

/// Reads the 32-byte vault key (hex) from the OS secure store, generating and
/// persisting one on first launch.
Future<String> _loadOrCreateKey() async {
  const secure = FlutterSecureStorage();
  final existing = await secure.read(key: _vaultKeyName);
  if (existing != null && existing.length == 64) return existing;

  final rng = Random.secure();
  final bytes = List<int>.generate(32, (_) => rng.nextInt(256));
  final hex = bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  await secure.write(key: _vaultKeyName, value: hex);
  return hex;
}

class StunnerApp extends StatefulWidget {
  const StunnerApp({super.key, required this.core, required this.store});

  final StunnerCore core;
  final ChatStore store;

  @override
  State<StunnerApp> createState() => _StunnerAppState();
}

class _StunnerAppState extends State<StunnerApp> with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.core.stop();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Save battery: stop polling in the background, resume on foreground.
    final source = widget.store; // store owns the source via bootstrap
    switch (state) {
      case AppLifecycleState.resumed:
        source.resumePolling();
      case AppLifecycleState.paused:
      case AppLifecycleState.inactive:
      case AppLifecycleState.hidden:
      case AppLifecycleState.detached:
        source.pausePolling();
    }
  }

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
      home: ChatsScreen(core: widget.core, store: widget.store),
    );
  }
}
