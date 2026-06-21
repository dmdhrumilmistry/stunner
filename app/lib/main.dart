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

class StunnerApp extends StatelessWidget {
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
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: appState,
      builder: (context, _) => MaterialApp(
        title: 'Stunner',
        debugShowCheckedModeBanner: false,
        theme: AppTheme.light(),
        darkTheme: AppTheme.dark(),
        themeMode: appState.themeMode,
        home: appState.onboarded
            ? HomeShell(
                core: core,
                store: store,
                appState: appState,
                notifications: notifications,
              )
            : OnboardingScreen(appState: appState, messaging: messaging),
      ),
    );
  }
}
