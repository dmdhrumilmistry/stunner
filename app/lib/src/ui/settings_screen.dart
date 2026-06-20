import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import 'my_identity_screen.dart';

/// Settings: network (STUN/TURN), security, and an honest note about the
/// current local-only build.
class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key, required this.core});

  final StunnerCore core;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool _relayEnabled = false;
  String _appLock = 'none';

  void _showIceServers() {
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('ICE servers (STUN/TURN)'),
        content: const Text(
          'Defaults:\n'
          '  • stun:stun.l.google.com:19302\n'
          '  • stun:stun1.l.google.com:19302\n\n'
          'Override these with your own STUN/TURN (e.g. a self-hosted coturn) '
          'in the core settings. Public STUN only assists discovery; for '
          'restrictive networks add a TURN server.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          Container(
            margin: const EdgeInsets.all(12),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              children: [
                const Icon(Icons.info_outline),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    'This build runs the encrypted core locally (identity, '
                    'verification, message UI). Live two-device delivery over '
                    'STUN/TURN is the next integration step.',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ),
              ],
            ),
          ),
          const _SectionHeader('Network (STUN / TURN)'),
          ListTile(
            leading: const Icon(Icons.dns_outlined),
            title: const Text('ICE servers'),
            subtitle: const Text('Public STUN defaults; override with your own'),
            trailing: const Icon(Icons.chevron_right),
            onTap: _showIceServers,
          ),
          SwitchListTile(
            secondary: const Icon(Icons.inbox_outlined),
            title: const Text('Offline relay (optional)'),
            subtitle: const Text(
              'Off by default. A self-hostable, content-blind mailbox for '
              'offline delivery.',
            ),
            value: _relayEnabled,
            onChanged: (v) => setState(() => _relayEnabled = v),
          ),
          const Divider(),
          const _SectionHeader('Security & Privacy'),
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: const Text('App lock'),
            subtitle: Text(_appLock == 'none' ? 'Disabled' : _appLock),
            trailing: DropdownButton<String>(
              value: _appLock,
              items: const [
                DropdownMenuItem(value: 'none', child: Text('None')),
                DropdownMenuItem(value: 'pin', child: Text('PIN')),
                DropdownMenuItem(value: 'biometric', child: Text('Biometric')),
              ],
              onChanged: (v) => setState(() => _appLock = v ?? 'none'),
            ),
          ),
          ListTile(
            leading: const Icon(Icons.qr_code_2),
            title: const Text('My identity & safety number'),
            subtitle: const Text('Show your QR code and verify a contact'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => MyIdentityScreen(core: widget.core),
              ),
            ),
          ),
          const Divider(),
          const _SectionHeader('About'),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('Core version'),
            subtitle: Text(widget.core.version()),
          ),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader(this.title);

  final String title;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        title.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: Theme.of(context).colorScheme.primary,
              letterSpacing: 1,
            ),
      ),
    );
  }
}
