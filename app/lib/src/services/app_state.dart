import 'package:flutter/material.dart';

import '../models/chat.dart' show initialsOf;

/// The local user's editable profile. Mirrors the fields the design's profile
/// editor exposes. Persisted to the encrypted Go account over FFI in a later
/// step; in-session only for now.
class Profile {
  Profile({
    required this.name,
    required this.username,
    required this.status,
    required this.email,
  });

  String name;
  String username;
  String status;
  String email;

  String get initials => initialsOf(name);
}

/// User preferences shown across Appearance / Notifications / Privacy. All are
/// effective in-session (e.g. enterToSend and reduceMotion change behavior).
class Prefs {
  bool readReceipts = true;
  bool typingIndicators = true;
  bool onlineStatus = true;
  bool lastSeen = false;

  bool notifSound = true;
  bool notifPreview = true;
  bool notifReactions = false;
  bool notifGroup = true;

  bool reduceMotion = false;
  bool enterToSend = true;
}

/// App-wide state: theme mode, the local user's profile, and preferences.
/// A [ChangeNotifier] so any screen can react to edits immediately.
class AppState extends ChangeNotifier {
  ThemeMode _themeMode = ThemeMode.system;
  final Profile profile = Profile(name: '', username: '', status: '', email: '');
  final Prefs prefs = Prefs();

  /// Whether first-launch onboarding has been completed this session.
  bool onboarded = false;

  /// This device's shareable contact URI (set at onboarding when the runtime
  /// starts). Share it so peers can add you.
  String myContactCode = '';

  ThemeMode get themeMode => _themeMode;

  /// Completes onboarding: stores the user-entered details and their contact
  /// code, then reveals the app.
  void completeOnboarding({
    required String name,
    String username = '',
    String status = '',
    String email = '',
    String contactCode = '',
  }) {
    profile
      ..name = name.trim()
      ..username = username.trim()
      ..status = status.trim().isEmpty ? 'Available' : status.trim()
      ..email = email.trim();
    myContactCode = contactCode;
    onboarded = true;
    notifyListeners();
  }

  bool isDark(BuildContext context) {
    switch (_themeMode) {
      case ThemeMode.dark:
        return true;
      case ThemeMode.light:
        return false;
      case ThemeMode.system:
        return MediaQuery.platformBrightnessOf(context) == Brightness.dark;
    }
  }

  void setThemeMode(ThemeMode mode) {
    if (_themeMode == mode) return;
    _themeMode = mode;
    notifyListeners();
  }

  /// Flips between explicit light and dark (resolving "system" first).
  void toggleTheme(BuildContext context) {
    setThemeMode(isDark(context) ? ThemeMode.light : ThemeMode.dark);
  }

  /// Applies edited profile fields and notifies listeners.
  void updateProfile({
    required String name,
    required String username,
    required String status,
    required String email,
  }) {
    profile
      ..name = name.trim().isEmpty ? profile.name : name.trim()
      ..username = username.trim()
      ..status = status.trim()
      ..email = email.trim();
    notifyListeners();
  }

  /// Mutates a single preference via [apply] and notifies listeners.
  void updatePrefs(void Function(Prefs p) apply) {
    apply(prefs);
    notifyListeners();
  }
}
