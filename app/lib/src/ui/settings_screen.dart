import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import 'ice_servers_screen.dart';
import 'my_identity_screen.dart';

/// Settings: STUN/TURN override, optional relay, app lock, and a diagnostics
/// section that exercises the Go core over FFI.
class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key, required this.core});

  final StunnerCore core;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool _relayEnabled = false;
  String _appLock = 'none';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          const _SectionHeader('Network (STUN / TURN)'),
          ListTile(
            leading: const Icon(Icons.dns_outlined),
            title: const Text('ICE servers'),
            subtitle: const Text(
              'Defaults use public STUN. Override with your own STUN/TURN '
              '(e.g. self-hosted coturn).',
            ),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => const IceServersScreen(),
              ),
            ),
          ),
          SwitchListTile(
            secondary: const Icon(Icons.inbox_outlined),
            title: const Text('Offline relay (optional)'),
            subtitle: const Text(
              'Off by default. When on, a self-hostable, content-blind mailbox '
              'holds encrypted messages for offline delivery.',
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
