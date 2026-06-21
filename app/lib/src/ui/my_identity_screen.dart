import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../ffi/stunner_ffi.dart';

/// Shows the user's contact QR code + ID (for a peer to scan or paste) and
/// computes the verification safety number against a pasted/scanned peer URI.
///
/// [myCode] is this device's stable contact URI from the running account. If it
/// is empty (runtime not started), a one-off code is generated for display.
class MyIdentityScreen extends StatefulWidget {
  const MyIdentityScreen({super.key, required this.core, this.myCode = ''});

  final StunnerCore core;
  final String myCode;

  @override
  State<MyIdentityScreen> createState() => _MyIdentityScreenState();
}

class _MyIdentityScreenState extends State<MyIdentityScreen> {
  late final String _myUri =
      widget.myCode.isNotEmpty ? widget.myCode : widget.core.newContactURI('me');
  final _peerController = TextEditingController();
  String? _safetyNumber;
  String? _error;

  @override
  void dispose() {
    _peerController.dispose();
    super.dispose();
  }

  void _verify() {
    setState(() {
      _error = null;
      _safetyNumber = null;
    });
    final peer = _peerController.text.trim();
    if (peer.isEmpty) return;
    try {
      widget.core.validateContactURI(peer); // throws on malformed input
      setState(() => _safetyNumber = widget.core.safetyNumber(_myUri, peer));
    } on FormatException catch (e) {
      setState(() => _error = 'Invalid contact code: ${e.message}');
    }
  }

  @override
  Widget build(BuildContext context) {
    final fp = widget.core.available
        ? widget.core.validateContactURI(_myUri).fingerprint
        : 'core unavailable';

    return Scaffold(
      appBar: AppBar(title: const Text('My identity')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Center(
            // Always render the QR as black-on-white so it stays scannable in
            // dark theme (default modules are black; a dark card behind them
            // made it unreadable).
            child: Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
              ),
              child: QrImageView(
                data: _myUri,
                size: 220,
                version: QrVersions.auto,
                backgroundColor: Colors.white,
                eyeStyle: const QrEyeStyle(
                  eyeShape: QrEyeShape.square,
                  color: Colors.black,
                ),
                dataModuleStyle: const QrDataModuleStyle(
                  dataModuleShape: QrDataModuleShape.square,
                  color: Colors.black,
                ),
              ),
            ),
          ),
          const SizedBox(height: 12),
          Text('Your contact ID', style: Theme.of(context).textTheme.labelMedium),
          SelectableText(_myUri, style: const TextStyle(fontSize: 12.5)),
          const SizedBox(height: 6),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: _myUri));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Contact ID copied')),
                );
              },
              icon: const Icon(Icons.copy, size: 16),
              label: const Text('Copy ID'),
            ),
          ),
          const SizedBox(height: 12),
          Text('Fingerprint', style: Theme.of(context).textTheme.labelMedium),
          SelectableText(fp),
          const SizedBox(height: 24),
          Text(
            'Verify a contact',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          const Text(
            'Paste a peer\'s contact code (or scan their QR), then compare the '
            'safety number on both devices. They must match exactly.',
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _peerController,
            decoration: const InputDecoration(
              labelText: 'Peer contact URI (stunner:contact?...)',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 8),
          FilledButton.icon(
            onPressed: _verify,
            icon: const Icon(Icons.verified_user_outlined),
            label: const Text('Compute safety number'),
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
          ],
          if (_safetyNumber != null) ...[
            const SizedBox(height: 16),
            Text('Safety number', style: Theme.of(context).textTheme.labelMedium),
            SelectableText(
              _safetyNumber!,
              style: const TextStyle(fontFeatures: [FontFeature.tabularFigures()]),
            ),
          ],
        ],
      ),
    );
  }
}
