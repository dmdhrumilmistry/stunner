import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Editor for the STUN/TURN (ICE) server list. Persists a JSON array to the OS
/// secure store; the value is read at launch and passed to the runtime, so
/// changes apply on next app start. Must match the key main.dart reads.
const iceServersStorageKey = 'stunner_ice_servers';

/// JSON array of `{"urls":[...],"username":"","credential":""}` — mirrors
/// core/pkg/settings.ICEServer.
class IceServersScreen extends StatefulWidget {
  const IceServersScreen({super.key});

  @override
  State<IceServersScreen> createState() => _IceServersScreenState();
}

class _IceServersScreenState extends State<IceServersScreen> {
  final _controller = TextEditingController();
  final _storage = const FlutterSecureStorage();
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final stored = await _storage.read(key: iceServersStorageKey);
    setState(() {
      _controller.text = (stored == null || stored.isEmpty)
          ? _placeholder
          : stored;
      _loading = false;
    });
  }

  static const _placeholder = '[\n'
      '  {"urls": ["stun:stun.l.google.com:19302"]}\n'
      ']';

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final text = _controller.text.trim();
    if (text.isNotEmpty) {
      try {
        final decoded = jsonDecode(text);
        if (decoded is! List) {
          setState(() => _error = 'Expected a JSON array of ICE servers.');
          return;
        }
      } on FormatException catch (e) {
        setState(() => _error = 'Invalid JSON: ${e.message}');
        return;
      }
    }
    await _storage.write(key: iceServersStorageKey, value: text);
    if (!mounted) return;
    setState(() => _error = null);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Saved. Restart the app to apply.')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ICE servers'),
        actions: [
          IconButton(
            tooltip: 'Save',
            icon: const Icon(Icons.save_outlined),
            onPressed: _loading ? null : _save,
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                const Text(
                  'STUN/TURN servers as a JSON array. Leave empty to use the '
                  'built-in public STUN defaults. For reliable NAT traversal, '
                  'point at your own coturn (TURN).',
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _controller,
                  maxLines: 12,
                  style: const TextStyle(fontFamily: 'monospace'),
                  decoration: const InputDecoration(
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(_error!,
                      style: TextStyle(color: Theme.of(context).colorScheme.error)),
                ],
                const SizedBox(height: 16),
                const Text(
                  'Example TURN entry:\n'
                  '{"urls": ["turn:turn.example.com:3478"], '
                  '"username": "user", "credential": "pass"}',
                  style: TextStyle(fontFamily: 'monospace', fontSize: 12),
                ),
              ],
            ),
    );
  }
}
