import 'package:flutter/material.dart';

import 'src/ffi/stunner_ffi.dart';
import 'src/services/app_state.dart';
import 'src/services/chat_store.dart';
import 'src/services/messaging_service.dart';
import 'src/services/notification_service.dart';
import 'src/theme/app_theme.dart';
import 'src/ui/home_shell.dart';
import 'src/ui/onboarding_screen.dart';

void main() {
  // Open the native Stunner core (FFI). Degrades gracefully if the library has
  // not been built yet.
  final core = StunnerCore.open();
  // ignore: avoid_print
  print('Stunner core: ${core.version()}');

  final notifications = NotificationService();
  final store = ChatStore(notifications: notifications);
  final appState = AppState();
  final messaging = MessagingService(core, store);

  runApp(StunnerApp(
    core: core,
    store: store,
    appState: appState,
    notifications: notifications,
    messaging: messaging,
  ));
}

class StunnerApp extends StatefulWidget {
  const StunnerApp({
    super.key,
    required this.core,
    required this.store,
    required this.appState,
    required this.notifications,
    required this.messaging,
  });

  final StunnerCore core;
  final ChatStore store;
  final AppState appState;
  final NotificationService notifications;
  final MessagingService messaging;

  @override
  State<StunnerApp> createState() => _StunnerAppState();
}

class _StunnerAppState extends State<StunnerApp> {
  late final Future<void> _boot = widget.messaging.bootstrap(widget.appState);

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.appState,
      builder: (context, _) => MaterialApp(
        title: 'Stunner',
        debugShowCheckedModeBanner: false,
        theme: AppTheme.light(),
        darkTheme: AppTheme.dark(),
        themeMode: widget.appState.themeMode,
        home: FutureBuilder<void>(
          future: _boot,
          builder: (context, snap) {
            if (snap.connectionState != ConnectionState.done) {
              return const Scaffold(body: Center(child: CircularProgressIndicator()));
            }
            return widget.appState.onboarded
                ? HomeShell(
                    core: widget.core,
                    store: widget.store,
                    appState: widget.appState,
                    notifications: widget.notifications,
                  )
                : OnboardingScreen(appState: widget.appState, messaging: widget.messaging);
          },
        ),
      ),
    );
  }
}
