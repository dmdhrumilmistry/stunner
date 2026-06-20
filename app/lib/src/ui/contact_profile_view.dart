import 'package:flutter/material.dart';

import '../services/chat_store.dart';
import '../theme/app_theme.dart';
import 'widgets.dart';

/// A contact's profile: avatar, presence, quick actions (Message / Call / Video),
/// contact details, and mute / block / delete. Reused embedded and pushed.
class ContactProfileView extends StatelessWidget {
  const ContactProfileView({
    super.key,
    required this.store,
    required this.contactId,
    this.showBack = false,
    this.onBack,
    required this.onMessage,
    this.onDeleted,
  });

  final ChatStore store;
  final String contactId;
  final bool showBack;
  final VoidCallback? onBack;
  final void Function(String contactId) onMessage;
  final VoidCallback? onDeleted;

  void _notice(BuildContext context, String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _confirmDelete(BuildContext context, String name) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Delete $name?'),
        content: const Text('This removes the contact from this device.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Delete')),
        ],
      ),
    );
    if (ok == true) {
      store.deleteContact(contactId);
      onDeleted?.call();
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: store,
      builder: (context, _) {
        final contact = store.contactById(contactId);
        if (contact == null) {
          return _NotFound(showBack: showBack, onBack: onBack);
        }
        final scheme = Theme.of(context).colorScheme;
        return Column(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
              child: Row(
                children: [
                  if (showBack)
                    IconButton(
                      icon: const Icon(Icons.arrow_back_ios_new, size: 20),
                      onPressed: onBack,
                    ),
                  const Spacer(),
                  IconButton(
                    icon: const Icon(Icons.more_horiz),
                    onPressed: () => _confirmDelete(context, contact.name),
                  ),
                ],
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(18, 4, 18, 28),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 520),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Column(
                          children: [
                            Avatar(
                              initials: contact.initials,
                              size: 96,
                              online: contact.online,
                              showDot: true,
                            ),
                            const SizedBox(height: 14),
                            Text(
                              contact.name,
                              textAlign: TextAlign.center,
                              style: const TextStyle(
                                fontSize: 23,
                                fontWeight: FontWeight.w700,
                                letterSpacing: -0.4,
                              ),
                            ),
                            if (contact.role.isNotEmpty) ...[
                              const SizedBox(height: 3),
                              Text(contact.role,
                                  style: TextStyle(fontSize: 14.5, color: scheme.onSurfaceVariant)),
                            ],
                            const SizedBox(height: 3),
                            Text(
                              contact.online ? 'Online' : 'Offline',
                              style: TextStyle(
                                fontSize: 13,
                                color: contact.online ? AppTheme.online : scheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 22),
                        Row(
                          children: [
                            _QuickAction(
                              icon: Icons.chat_bubble_outline,
                              label: 'Message',
                              onTap: () => onMessage(contact.id),
                            ),
                            const SizedBox(width: 10),
                            _QuickAction(
                              icon: Icons.call_outlined,
                              label: 'Call',
                              onTap: () => _notice(context, 'Calls need a live connection (coming soon).'),
                            ),
                            const SizedBox(width: 10),
                            _QuickAction(
                              icon: Icons.videocam_outlined,
                              label: 'Video',
                              onTap: () => _notice(context, 'Video needs a live connection (coming soon).'),
                            ),
                          ],
                        ),
                        const SizedBox(height: 22),
                        SettingsCard(children: [
                          if (contact.email.isNotEmpty)
                            _DetailRow(icon: Icons.mail_outline, label: 'Email', value: contact.email),
                          if (contact.phone.isNotEmpty)
                            _DetailRow(icon: Icons.call_outlined, label: 'Phone', value: contact.phone),
                          if (contact.fingerprint.isNotEmpty)
                            _DetailRow(
                              icon: Icons.fingerprint,
                              label: 'Fingerprint',
                              value: contact.fingerprint,
                            ),
                          if (contact.email.isEmpty &&
                              contact.phone.isEmpty &&
                              contact.fingerprint.isEmpty)
                            const _DetailRow(
                              icon: Icons.lock_outline,
                              label: 'Encryption',
                              value: 'End-to-end encrypted',
                            ),
                        ]),
                        const SizedBox(height: 16),
                        SettingsCard(children: [
                          _ActionRow(
                            icon: contact.muted ? Icons.notifications_off : Icons.notifications_outlined,
                            label: contact.muted ? 'Unmute notifications' : 'Mute notifications',
                            onTap: () {
                              store.toggleMute(contact.id);
                              _notice(context, contact.muted ? 'Unmuted' : 'Muted');
                            },
                          ),
                          _ActionRow(
                            icon: Icons.block,
                            label: contact.blocked ? 'Unblock contact' : 'Block contact',
                            destructive: true,
                            onTap: () {
                              store.toggleBlock(contact.id);
                              _notice(context, contact.blocked ? 'Blocked' : 'Unblocked');
                            },
                          ),
                          _ActionRow(
                            icon: Icons.delete_outline,
                            label: 'Delete contact',
                            destructive: true,
                            onTap: () => _confirmDelete(context, contact.name),
                          ),
                        ]),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}

class _QuickAction extends StatelessWidget {
  const _QuickAction({required this.icon, required this.label, required this.onTap});

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Expanded(
      child: OutlinedButton(
        onPressed: onTap,
        style: OutlinedButton.styleFrom(
          padding: const EdgeInsets.symmetric(vertical: 12),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 20, color: scheme.onSurface),
            const SizedBox(height: 6),
            Text(label, style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w500)),
          ],
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({required this.icon, required this.label, required this.value});

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 19, color: scheme.onSurfaceVariant),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: TextStyle(fontSize: 12, color: scheme.onSurfaceVariant)),
                const SizedBox(height: 1),
                SelectableText(value, style: const TextStyle(fontSize: 14.5)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionRow extends StatelessWidget {
  const _ActionRow({
    required this.icon,
    required this.label,
    required this.onTap,
    this.destructive = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool destructive;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = destructive ? scheme.error : scheme.onSurface;
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        child: Row(
          children: [
            Icon(icon, size: 19, color: destructive ? scheme.error : scheme.onSurfaceVariant),
            const SizedBox(width: 12),
            Text(label, style: TextStyle(fontSize: 14.5, color: color)),
          ],
        ),
      ),
    );
  }
}

class _NotFound extends StatelessWidget {
  const _NotFound({required this.showBack, required this.onBack});

  final bool showBack;
  final VoidCallback? onBack;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        if (showBack)
          Align(
            alignment: Alignment.centerLeft,
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: IconButton(
                icon: const Icon(Icons.arrow_back_ios_new, size: 20),
                onPressed: onBack,
              ),
            ),
          ),
        const Expanded(child: Center(child: Text('Contact not found'))),
      ],
    );
  }
}
