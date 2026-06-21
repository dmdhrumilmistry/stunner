import 'package:flutter/material.dart';

import '../services/app_state.dart';
import '../services/messaging_service.dart';

/// First-launch onboarding: the user enters their own profile details manually,
/// and we boot the live messaging runtime to obtain their shareable contact ID.
class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({
    super.key,
    required this.appState,
    required this.messaging,
  });

  final AppState appState;
  final MessagingService messaging;

  @override
  State<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends State<OnboardingScreen> {
  final _name = TextEditingController();
  final _username = TextEditingController();
  final _status = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _username.dispose();
    _status.dispose();
    super.dispose();
  }

  Future<void> _continue() async {
    final name = _name.text.trim();
    if (name.isEmpty) {
      setState(() => _error = 'Enter a display name.');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });

    // Boot the runtime to get this device's contact code. If the native core
    // isn't available, continue in degraded mode (no live messaging).
    final res = await widget.messaging.start(name);
    if (!mounted) return;

    widget.appState.completeOnboarding(
      name: name,
      username: _username.text,
      status: _status.text,
      contactCode: res.uri,
    );
    if (!res.ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Messaging unavailable: ${res.error}')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 64,
                    height: 64,
                    decoration: BoxDecoration(
                      color: scheme.primary,
                      borderRadius: BorderRadius.circular(18),
                    ),
                    child: Icon(Icons.chat_bubble, color: scheme.onPrimary, size: 32),
                  ),
                  const SizedBox(height: 20),
                  const Text('Welcome to Stunner',
                      textAlign: TextAlign.center,
                      style: TextStyle(fontSize: 24, fontWeight: FontWeight.w700, letterSpacing: -0.5)),
                  const SizedBox(height: 6),
                  Text(
                    'Set up your profile. Your identity is generated on this '
                    'device — share your contact ID so others can message you.',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 13.5, height: 1.4, color: scheme.onSurfaceVariant),
                  ),
                  const SizedBox(height: 28),
                  _field('Display name', _name, autofocus: true),
                  const SizedBox(height: 16),
                  _field('Username (optional)', _username),
                  const SizedBox(height: 16),
                  _field('Status (optional)', _status, hint: 'Available'),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!, style: TextStyle(color: scheme.error)),
                  ],
                  const SizedBox(height: 24),
                  FilledButton(
                    onPressed: _busy ? null : _continue,
                    style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
                    child: _busy
                        ? const SizedBox(
                            width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Get started'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _field(String label, TextEditingController c, {bool autofocus = false, String? hint}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 2, bottom: 7),
          child: Text(label, style: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w500)),
        ),
        TextField(
          controller: c,
          autofocus: autofocus,
          decoration: InputDecoration(hintText: hint),
        ),
      ],
    );
  }
}
