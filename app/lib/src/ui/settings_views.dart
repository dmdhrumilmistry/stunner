import 'dart:convert';

import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import '../services/app_state.dart';
import '../theme/app_theme.dart';
import 'my_identity_screen.dart';
import 'widgets.dart';

/// Edit the local user's own profile (display name, username, status, email).
/// This is the "view & update your profile" feature; edits flow into [AppState]
/// and are reflected everywhere immediately.
class ProfileEditView extends StatefulWidget {
  const ProfileEditView({super.key, required this.appState, required this.core});

  final AppState appState;
  final StunnerCore core;

  @override
  State<ProfileEditView> createState() => _ProfileEditViewState();
}

class _ProfileEditViewState extends State<ProfileEditView> {
  late final TextEditingController _name;
  late final TextEditingController _username;
  late final TextEditingController _status;
  late final TextEditingController _email;
  bool _saved = false;

  @override
  void initState() {
    super.initState();
    final p = widget.appState.profile;
    _name = TextEditingController(text: p.name);
    _username = TextEditingController(text: p.username);
    _status = TextEditingController(text: p.status);
    _email = TextEditingController(text: p.email);
  }

  @override
  void dispose() {
    _name.dispose();
    _username.dispose();
    _status.dispose();
    _email.dispose();
    super.dispose();
  }

  void _save() {
    widget.appState.updateProfile(
      name: _name.text,
      username: _username.text,
      status: _status.text,
      email: _email.text,
    );
    setState(() => _saved = true);
    Future.delayed(const Duration(seconds: 2), () {
      if (mounted) setState(() => _saved = false);
    });
    FocusScope.of(context).unfocus();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return ListenableBuilder(
      listenable: widget.appState,
      builder: (context, _) {
        return ListView(
          padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
          children: [
            Center(
              child: Stack(
                clipBehavior: Clip.none,
                children: [
                  Avatar(initials: widget.appState.profile.initials, size: 96),
                  Positioned(
                    right: -2,
                    bottom: -2,
                    child: Material(
                      color: scheme.primary,
                      shape: const CircleBorder(),
                      child: InkWell(
                        customBorder: const CircleBorder(),
                        onTap: () => ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Custom avatars need a live build (coming soon).')),
                        ),
                        child: Padding(
                          padding: const EdgeInsets.all(8),
                          child: Icon(Icons.camera_alt_outlined, size: 16, color: scheme.onPrimary),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 28),
            _Field(label: 'Display name', controller: _name),
            const SizedBox(height: 16),
            _Field(label: 'Username', controller: _username),
            const SizedBox(height: 16),
            _Field(label: 'Status', controller: _status),
            const SizedBox(height: 16),
            _Field(label: 'Email', controller: _email, keyboardType: TextInputType.emailAddress),
            const SizedBox(height: 22),
            Row(
              children: [
                FilledButton(onPressed: _save, child: const Text('Save changes')),
                const SizedBox(width: 12),
                if (_saved)
                  const Row(
                    children: [
                      Icon(Icons.check, size: 16, color: AppTheme.online),
                      SizedBox(width: 4),
                      Text('Saved',
                          style: TextStyle(
                              color: AppTheme.online, fontWeight: FontWeight.w500, fontSize: 13.5)),
                    ],
                  ),
              ],
            ),
          ],
        );
      },
    );
  }
}

class _Field extends StatelessWidget {
  const _Field({required this.label, required this.controller, this.keyboardType});

  final String label;
  final TextEditingController controller;
  final TextInputType? keyboardType;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 2, bottom: 7),
          child: Text(label, style: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w500)),
        ),
        TextField(controller: controller, keyboardType: keyboardType),
      ],
    );
  }
}

/// Appearance: theme selector + motion/behavior toggles.
class AppearanceView extends StatelessWidget {
  const AppearanceView({super.key, required this.appState});

  final AppState appState;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: appState,
      builder: (context, _) {
        final dark = appState.isDark(context);
        return ListView(
          padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
          children: [
            const SectionLabel('Theme'),
            _ThemeSegment(
              dark: dark,
              onLight: () => appState.setThemeMode(ThemeMode.light),
              onDark: () => appState.setThemeMode(ThemeMode.dark),
            ),
            const SizedBox(height: 24),
            SettingsCard(children: [
              _ToggleRow(
                label: 'Reduce motion',
                desc: 'Minimize animations across the app',
                value: appState.prefs.reduceMotion,
                onChanged: (v) => appState.updatePrefs((p) => p.reduceMotion = v),
              ),
              _ToggleRow(
                label: 'Enter to send',
                desc: 'Press Enter to send a message',
                value: appState.prefs.enterToSend,
                onChanged: (v) => appState.updatePrefs((p) => p.enterToSend = v),
              ),
            ]),
          ],
        );
      },
    );
  }
}

class _ThemeSegment extends StatelessWidget {
  const _ThemeSegment({required this.dark, required this.onLight, required this.onDark});

  final bool dark;
  final VoidCallback onLight;
  final VoidCallback onDark;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    Widget seg(String label, IconData icon, bool selected, VoidCallback onTap) {
      return Expanded(
        child: GestureDetector(
          onTap: onTap,
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 10),
            decoration: BoxDecoration(
              color: selected ? scheme.surface : Colors.transparent,
              borderRadius: BorderRadius.circular(10),
              boxShadow: selected
                  ? [BoxShadow(color: Colors.black.withValues(alpha: 0.06), blurRadius: 4)]
                  : null,
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, size: 17),
                const SizedBox(width: 6),
                Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
              ],
            ),
          ),
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.all(5),
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: BorderRadius.circular(13),
      ),
      child: Row(
        children: [
          seg('Light', Icons.light_mode_outlined, !dark, onLight),
          const SizedBox(width: 6),
          seg('Dark', Icons.dark_mode_outlined, dark, onDark),
        ],
      ),
    );
  }
}

/// Notifications preferences.
class NotificationsView extends StatelessWidget {
  const NotificationsView({super.key, required this.appState});

  final AppState appState;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: appState,
      builder: (context, _) {
        final p = appState.prefs;
        return ListView(
          padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
          children: [
            SettingsCard(children: [
              _ToggleRow(
                label: 'Sound',
                desc: 'Play a sound for new messages',
                value: p.notifSound,
                onChanged: (v) => appState.updatePrefs((x) => x.notifSound = v),
              ),
              _ToggleRow(
                label: 'Message preview',
                desc: 'Show message text in notifications',
                value: p.notifPreview,
                onChanged: (v) => appState.updatePrefs((x) => x.notifPreview = v),
              ),
              _ToggleRow(
                label: 'Reactions',
                desc: 'Notify when someone reacts',
                value: p.notifReactions,
                onChanged: (v) => appState.updatePrefs((x) => x.notifReactions = v),
              ),
              _ToggleRow(
                label: 'Group messages',
                desc: 'Notify for group activity',
                value: p.notifGroup,
                onChanged: (v) => appState.updatePrefs((x) => x.notifGroup = v),
              ),
            ]),
          ],
        );
      },
    );
  }
}

/// Network diagnostics: run a real STUN reachability probe via the Go core and
/// show whether a public (server-reflexive) address could be discovered.
class NetworkView extends StatefulWidget {
  const NetworkView({super.key, required this.core});

  final StunnerCore core;

  @override
  State<NetworkView> createState() => _NetworkViewState();
}

class _NetworkViewState extends State<NetworkView> {
  bool _running = false;
  StunResult? _result;

  /// The full settings map (so we preserve non-ICE fields on save).
  Map<String, dynamic> _settings = {};

  @override
  void initState() {
    super.initState();
    _loadSettings();
  }

  void _loadSettings() {
    if (!widget.core.available) return;
    try {
      _settings = jsonDecode(widget.core.getSettings()) as Map<String, dynamic>;
    } on Object {
      _settings = {};
    }
  }

  List<Map<String, dynamic>> get _iceServers {
    final raw = _settings['iceServers'];
    if (raw is List) {
      return raw.whereType<Map>().map((e) => e.cast<String, dynamic>()).toList();
    }
    return [];
  }

  List<String> _urlsOf(Map<String, dynamic> server) {
    final u = server['urls'];
    if (u is List) return u.whereType<String>().toList();
    if (u is String) return [u];
    return [];
  }

  void _saveServers(List<Map<String, dynamic>> servers) {
    _settings['iceServers'] = servers;
    widget.core.setSettings(jsonEncode(_settings));
    setState(() {});
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Saved — applies after restarting the app.')),
      );
    }
  }

  Future<void> _addServer() async {
    final urlCtl = TextEditingController();
    final userCtl = TextEditingController();
    final credCtl = TextEditingController();
    final added = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add STUN/TURN server'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: urlCtl,
              autofocus: true,
              decoration: const InputDecoration(
                labelText: 'URL',
                hintText: 'turn:turn.example.com:3478',
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: userCtl,
              decoration: const InputDecoration(labelText: 'Username (TURN, optional)'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: credCtl,
              decoration: const InputDecoration(labelText: 'Credential (TURN, optional)'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Add')),
        ],
      ),
    );
    if (added != true) return;
    final url = urlCtl.text.trim();
    if (url.isEmpty) return;
    final server = <String, dynamic>{
      'urls': [url],
      if (userCtl.text.trim().isNotEmpty) 'username': userCtl.text.trim(),
      if (credCtl.text.trim().isNotEmpty) 'credential': credCtl.text.trim(),
    };
    _saveServers([..._iceServers, server]);
  }

  void _removeServer(int index) {
    final servers = _iceServers;
    if (index < 0 || index >= servers.length) return;
    servers.removeAt(index);
    _saveServers(servers);
  }

  Future<void> _test() async {
    setState(() {
      _running = true;
      _result = null;
    });
    final res = await widget.core.checkStun();
    if (!mounted) return;
    setState(() {
      _running = false;
      _result = res;
    });
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final result = _result;
    return ListView(
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
      children: [
        const SectionLabel('Connectivity'),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: scheme.surfaceContainer,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('STUN connection',
                  style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
              const SizedBox(height: 4),
              Text(
                'Checks whether the default STUN servers can discover your public '
                'address — needed for direct peer-to-peer connections.',
                style: TextStyle(fontSize: 12.5, height: 1.4, color: scheme.onSurfaceVariant),
              ),
              const SizedBox(height: 14),
              if (result != null) _resultBanner(context, result),
              if (result != null) const SizedBox(height: 14),
              FilledButton.icon(
                onPressed: _running ? null : _test,
                icon: _running
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.wifi_find_outlined, size: 18),
                label: Text(_running ? 'Testing…' : 'Test STUN connection'),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        const SectionLabel('ICE servers'),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: scheme.surfaceContainer,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (_iceServers.isEmpty)
                const _IceLine('Using built-in public STUN defaults')
              else
                for (var i = 0; i < _iceServers.length; i++)
                  _IceLine(
                    _urlsOf(_iceServers[i]).join(', '),
                    isTurn: _urlsOf(_iceServers[i]).any((u) => u.startsWith('turn')),
                    onRemove: () => _removeServer(i),
                  ),
              const SizedBox(height: 8),
              Text(
                'Pure P2P uses STUN only for discovery; no message data passes '
                'through it. Add a self-hosted TURN server for restrictive networks. '
                'Changes apply after restarting the app.',
                style: TextStyle(fontSize: 12, height: 1.4, color: scheme.onSurfaceVariant),
              ),
              const SizedBox(height: 10),
              OutlinedButton.icon(
                onPressed: widget.core.available ? _addServer : null,
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Add STUN/TURN server'),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _resultBanner(BuildContext context, StunResult r) {
    final scheme = Theme.of(context).colorScheme;
    final good = r.ok;
    final color = good ? AppTheme.online : scheme.error;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withValues(alpha: 0.4)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(good ? Icons.check_circle_outline : Icons.error_outline, size: 20, color: color),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(good ? 'STUN reachable' : 'STUN unreachable',
                    style: TextStyle(fontWeight: FontWeight.w600, color: color)),
                const SizedBox(height: 2),
                Text(r.detail, style: const TextStyle(fontSize: 12.5, height: 1.4)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _IceLine extends StatelessWidget {
  const _IceLine(this.url, {this.isTurn = false, this.onRemove});

  final String url;
  final bool isTurn;
  final VoidCallback? onRemove;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Icon(isTurn ? Icons.swap_horiz : Icons.dns_outlined, size: 16, color: scheme.onSurfaceVariant),
          const SizedBox(width: 8),
          Expanded(
            child: Text(url,
                style: const TextStyle(fontSize: 13, fontFamily: 'monospace')),
          ),
          if (onRemove != null)
            IconButton(
              tooltip: 'Remove',
              visualDensity: VisualDensity.compact,
              icon: const Icon(Icons.close, size: 16),
              onPressed: onRemove,
            ),
        ],
      ),
    );
  }
}

/// Privacy & safety preferences, plus a link to the identity / safety-number
/// verification screen backed by the Go core.
class PrivacyView extends StatelessWidget {
  const PrivacyView({super.key, required this.appState, required this.core});

  final AppState appState;
  final StunnerCore core;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: appState,
      builder: (context, _) {
        final p = appState.prefs;
        final scheme = Theme.of(context).colorScheme;
        return ListView(
          padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
          children: [
            SettingsCard(children: [
              _ToggleRow(
                label: 'Read receipts',
                desc: 'Let others see when you have read messages',
                value: p.readReceipts,
                onChanged: (v) => appState.updatePrefs((x) => x.readReceipts = v),
              ),
              _ToggleRow(
                label: 'Typing indicators',
                desc: 'Show when you are typing',
                value: p.typingIndicators,
                onChanged: (v) => appState.updatePrefs((x) => x.typingIndicators = v),
              ),
              _ToggleRow(
                label: 'Online status',
                desc: 'Show when you are online',
                value: p.onlineStatus,
                onChanged: (v) => appState.updatePrefs((x) => x.onlineStatus = v),
              ),
              _ToggleRow(
                label: 'Last seen',
                desc: 'Share your last-active time',
                value: p.lastSeen,
                onChanged: (v) => appState.updatePrefs((x) => x.lastSeen = v),
              ),
            ]),
            const SizedBox(height: 16),
            SettingsCard(children: [
              InkWell(
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                      builder: (_) => MyIdentityScreen(core: core, myCode: appState.myContactCode)),
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
                  child: Row(
                    children: [
                      Icon(Icons.qr_code_2, size: 19, color: scheme.onSurfaceVariant),
                      const SizedBox(width: 12),
                      const Expanded(
                        child: Text('Identity & safety number',
                            style: TextStyle(fontSize: 14.5)),
                      ),
                      Icon(Icons.chevron_right, color: scheme.onSurfaceVariant),
                    ],
                  ),
                ),
              ),
            ]),
            Padding(
              padding: const EdgeInsets.fromLTRB(4, 12, 4, 0),
              child: Text(
                'These controls affect how you appear to others. Read receipts and '
                'typing indicators are reciprocal — turning them off hides others’ from you too.',
                style: TextStyle(fontSize: 12.5, height: 1.5, color: scheme.onSurfaceVariant),
              ),
            ),
          ],
        );
      },
    );
  }
}

class _ToggleRow extends StatelessWidget {
  const _ToggleRow({
    required this.label,
    required this.desc,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final String desc;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 12, 12),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                Text(desc, style: TextStyle(fontSize: 12.5, color: scheme.onSurfaceVariant)),
              ],
            ),
          ),
          Switch(value: value, onChanged: onChanged),
        ],
      ),
    );
  }
}
