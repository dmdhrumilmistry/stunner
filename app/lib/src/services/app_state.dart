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

  Map<String, dynamic> toMap() => {
        'readReceipts': readReceipts,
        'typingIndicators': typingIndicators,
        'onlineStatus': onlineStatus,
        'lastSeen': lastSeen,
        'notifSound': notifSound,
        'notifPreview': notifPreview,
        'notifReactions': notifReactions,
        'notifGroup': notifGroup,
        'reduceMotion': reduceMotion,
        'enterToSend': enterToSend,
      };

  void restoreFromMap(Map<String, dynamic> m) {
    readReceipts = m['readReceipts'] as bool? ?? readReceipts;
    typingIndicators = m['typingIndicators'] as bool? ?? typingIndicators;
    onlineStatus = m['onlineStatus'] as bool? ?? onlineStatus;
    lastSeen = m['lastSeen'] as bool? ?? lastSeen;
    notifSound = m['notifSound'] as bool? ?? notifSound;
    notifPreview = m['notifPreview'] as bool? ?? notifPreview;
    notifReactions = m['notifReactions'] as bool? ?? notifReactions;
    notifGroup = m['notifGroup'] as bool? ?? notifGroup;
    reduceMotion = m['reduceMotion'] as bool? ?? reduceMotion;
    enterToSend = m['enterToSend'] as bool? ?? enterToSend;
  }
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

  /// Serializes profile + prefs + onboarding for encrypted persistence.
  Map<String, dynamic> toMap() => {
        'onboarded': onboarded,
        'myContactCode': myContactCode,
        'themeMode': _themeMode.name,
        'profile': {
          'name': profile.name,
          'username': profile.username,
          'status': profile.status,
          'email': profile.email,
        },
        'prefs': prefs.toMap(),
      };

  /// Restores from a previously serialized map. Does not notify.
  void restoreFromMap(Map<String, dynamic> m) {
    onboarded = m['onboarded'] as bool? ?? false;
    myContactCode = m['myContactCode'] as String? ?? '';
    _themeMode = ThemeMode.values.asNameMap()[m['themeMode']] ?? ThemeMode.system;
    final p = (m['profile'] as Map?)?.cast<String, dynamic>() ?? const {};
    profile
      ..name = p['name'] as String? ?? ''
      ..username = p['username'] as String? ?? ''
      ..status = p['status'] as String? ?? ''
      ..email = p['email'] as String? ?? '';
    final pr = (m['prefs'] as Map?)?.cast<String, dynamic>();
    if (pr != null) prefs.restoreFromMap(pr);
  }

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
