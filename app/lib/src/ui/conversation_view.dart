import 'package:flutter/material.dart';

import '../models/chat.dart';
import '../services/app_state.dart';
import '../services/chat_store.dart';
import '../services/emoji.dart';
import '../theme/app_theme.dart';
import 'widgets.dart';

/// A single conversation: a header (presence + call/video), message bubbles with
/// reactions and read-receipt ticks, a typing indicator, and a working composer
/// (emoji, attach, send). Reused embedded (wide layout) and pushed (narrow).
class ConversationView extends StatefulWidget {
  const ConversationView({
    super.key,
    required this.store,
    required this.appState,
    required this.chatId,
    this.showBack = false,
    this.onBack,
    this.onOpenContact,
  });

  final ChatStore store;
  final AppState appState;
  final String chatId;
  final bool showBack;
  final VoidCallback? onBack;
  final void Function(String contactId)? onOpenContact;

  @override
  State<ConversationView> createState() => _ConversationViewState();
}

class _ConversationViewState extends State<ConversationView> {
  final _controller = TextEditingController();
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      widget.store.markRead(widget.chatId);
      _scrollToBottom(animate: false);
    });
  }

  @override
  void didUpdateWidget(ConversationView old) {
    super.didUpdateWidget(old);
    if (old.chatId != widget.chatId) {
      _controller.clear();
      WidgetsBinding.instance.addPostFrameCallback((_) {
        widget.store.markRead(widget.chatId);
        _scrollToBottom(animate: false);
      });
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _send() {
    final text = expandShortcodes(_controller.text);
    if (text.trim().isEmpty) return;
    widget.store.sendText(widget.chatId, text);
    _controller.clear();
    WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToBottom());
  }

  void _scrollToBottom({bool animate = true}) {
    if (!_scroll.hasClients) return;
    final target = _scroll.position.maxScrollExtent;
    if (animate && !widget.appState.prefs.reduceMotion) {
      _scroll.animateTo(target, duration: const Duration(milliseconds: 200), curve: Curves.easeOut);
    } else {
      _scroll.jumpTo(target);
    }
  }

  void _notice(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _messageActions(Message m) async {
    final action = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Wrap(
                spacing: 6,
                children: [
                  for (final e in const ['👍', '❤️', '😂', '🎉', '🙏', '🔥'])
                    IconButton(
                      onPressed: () => Navigator.pop(ctx, 'react:$e'),
                      icon: Text(e, style: const TextStyle(fontSize: 24)),
                    ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.delete_outline),
              title: const Text('Delete message'),
              onTap: () => Navigator.pop(ctx, 'delete'),
            ),
          ],
        ),
      ),
    );
    if (action == null) return;
    if (action == 'delete') {
      widget.store.deleteMessage(widget.chatId, m.id);
    } else if (action.startsWith('react:')) {
      widget.store.toggleReaction(widget.chatId, m.id, action.substring(6));
    }
  }

  Future<void> _pickEmoji() async {
    final picked = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) => SafeArea(
        child: GridView.count(
          crossAxisCount: 8,
          shrinkWrap: true,
          padding: const EdgeInsets.all(8),
          children: [
            for (final e in pickerEmojis)
              IconButton(
                onPressed: () => Navigator.pop(ctx, e),
                icon: Text(e, style: const TextStyle(fontSize: 22)),
              ),
          ],
        ),
      ),
    );
    if (picked == null) return;
    final sel = _controller.selection;
    final text = _controller.text;
    if (sel.isValid) {
      _controller.text = text.replaceRange(sel.start, sel.end, picked);
      _controller.selection = TextSelection.collapsed(offset: sel.start + picked.length);
    } else {
      _controller.text = text + picked;
      _controller.selection = TextSelection.collapsed(offset: _controller.text.length);
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.store,
      builder: (context, _) {
        final chat = widget.store.maybeChat(widget.chatId);
        if (chat == null) {
          return const Center(child: Text('Conversation not found'));
        }
        // Keep the open conversation read as new messages arrive, and follow
        // them to the bottom.
        if (chat.unread > 0) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            widget.store.markRead(widget.chatId);
            _scrollToBottom();
          });
        }
        final contact = widget.store.contactForChat(chat);
        return Column(
          children: [
            _header(context, chat, contact),
            Divider(height: 1, color: Theme.of(context).colorScheme.outline),
            Expanded(child: _messageList(chat, contact)),
            SafeArea(top: false, child: _composer(context)),
          ],
        );
      },
    );
  }

  Widget _header(BuildContext context, Chat chat, Contact? contact) {
    final scheme = Theme.of(context).colorScheme;
    final online = contact?.online ?? false;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
      constraints: const BoxConstraints(minHeight: 60),
      child: Row(
        children: [
          if (widget.showBack)
            IconButton(
              icon: const Icon(Icons.arrow_back_ios_new, size: 20),
              onPressed: widget.onBack,
            ),
          Expanded(
            child: InkWell(
              borderRadius: BorderRadius.circular(10),
              onTap: contact == null ? null : () => widget.onOpenContact?.call(contact.id),
              child: Padding(
                padding: const EdgeInsets.all(4),
                child: Row(
                  children: [
                    Avatar(initials: contact?.initials ?? '?', size: 42, online: online, showDot: true),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            chat.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 15.5, fontWeight: FontWeight.w600),
                          ),
                          Text(
                            online ? 'Online' : 'Offline',
                            style: TextStyle(
                              fontSize: 12.5,
                              color: online ? AppTheme.online : scheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
          IconButton(
            tooltip: 'Voice call',
            icon: const Icon(Icons.call_outlined),
            onPressed: () => _notice('Calls need a live connection (coming soon).'),
          ),
          IconButton(
            tooltip: 'Video call',
            icon: const Icon(Icons.videocam_outlined),
            onPressed: () => _notice('Video needs a live connection (coming soon).'),
          ),
        ],
      ),
    );
  }

  Widget _messageList(Chat chat, Contact? contact) {
    final messages = chat.messages;
    if (messages.isEmpty) {
      return const Center(child: Text('Say hello 👋'));
    }
    return ListView.builder(
      controller: _scroll,
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
      itemCount: messages.length,
      itemBuilder: (context, i) {
        final m = messages[i];
        return Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 760),
            child: GestureDetector(
              onLongPress: () => _messageActions(m),
              child: _Bubble(
                message: m,
                onReactionTap: (emoji) => widget.store.toggleReaction(widget.chatId, m.id, emoji),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _composer(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Expanded(
                child: Container(
                  decoration: BoxDecoration(
                    border: Border.all(color: scheme.outline),
                    borderRadius: BorderRadius.circular(24),
                    color: scheme.surface,
                  ),
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  child: Row(
                    children: [
                      IconButton(
                        tooltip: 'Attach',
                        icon: const Icon(Icons.attach_file, size: 20),
                        onPressed: () =>
                            _notice('File sharing needs a live connection (coming soon).'),
                      ),
                      Expanded(
                        child: TextField(
                          controller: _controller,
                          minLines: 1,
                          maxLines: 5,
                          textInputAction: widget.appState.prefs.enterToSend
                              ? TextInputAction.send
                              : TextInputAction.newline,
                          decoration: const InputDecoration(
                            hintText: 'Message',
                            filled: false,
                            border: InputBorder.none,
                            enabledBorder: InputBorder.none,
                            focusedBorder: InputBorder.none,
                            isDense: true,
                            contentPadding: EdgeInsets.symmetric(vertical: 10),
                          ),
                          onSubmitted: widget.appState.prefs.enterToSend ? (_) => _send() : null,
                        ),
                      ),
                      IconButton(
                        tooltip: 'Emoji',
                        icon: const Icon(Icons.emoji_emotions_outlined, size: 20),
                        onPressed: _pickEmoji,
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 48,
                height: 48,
                child: FilledButton(
                  onPressed: _send,
                  style: FilledButton.styleFrom(
                    padding: EdgeInsets.zero,
                    shape: const CircleBorder(),
                  ),
                  child: const Icon(Icons.arrow_upward, size: 22),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.message, required this.onReactionTap});

  final Message message;
  final void Function(String emoji) onReactionTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final me = message.fromMe;
    final radius = me
        ? const BorderRadius.only(
            topLeft: Radius.circular(18),
            topRight: Radius.circular(18),
            bottomLeft: Radius.circular(18),
            bottomRight: Radius.circular(6),
          )
        : const BorderRadius.only(
            topLeft: Radius.circular(18),
            topRight: Radius.circular(18),
            bottomLeft: Radius.circular(6),
            bottomRight: Radius.circular(18),
          );
    return Align(
      alignment: me ? Alignment.centerRight : Alignment.centerLeft,
      child: Column(
        crossAxisAlignment: me ? CrossAxisAlignment.end : CrossAxisAlignment.start,
        children: [
          Container(
            margin: const EdgeInsets.symmetric(vertical: 3),
            padding: const EdgeInsets.fromLTRB(14, 9, 14, 8),
            constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.74),
            decoration: BoxDecoration(
              color: me ? scheme.primary : scheme.surfaceContainerHighest,
              borderRadius: radius,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  message.text,
                  style: TextStyle(
                    fontSize: 15,
                    height: 1.3,
                    color: me ? scheme.onPrimary : scheme.onSurface,
                  ),
                ),
                const SizedBox(height: 2),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      _formatTime(message.time),
                      style: TextStyle(
                        fontSize: 11,
                        color: (me ? scheme.onPrimary : scheme.onSurfaceVariant)
                            .withValues(alpha: 0.7),
                      ),
                    ),
                    if (me) ...[
                      const SizedBox(width: 4),
                      _ReceiptTick(status: message.status, onPrimary: scheme.onPrimary),
                    ],
                  ],
                ),
              ],
            ),
          ),
          if (message.reactions.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Wrap(
                spacing: 4,
                children: [
                  for (final entry in message.reactions.entries)
                    InkWell(
                      borderRadius: BorderRadius.circular(99),
                      onTap: () => onReactionTap(entry.key),
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: scheme.surfaceContainerHighest,
                          borderRadius: BorderRadius.circular(99),
                          border: Border.all(color: scheme.outline),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(entry.key, style: const TextStyle(fontSize: 12)),
                            const SizedBox(width: 3),
                            Text('${entry.value}',
                                style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600)),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _ReceiptTick extends StatelessWidget {
  const _ReceiptTick({required this.status, required this.onPrimary});

  final DeliveryStatus status;
  final Color onPrimary;

  @override
  Widget build(BuildContext context) {
    final muted = onPrimary.withValues(alpha: 0.7);
    switch (status) {
      case DeliveryStatus.sending:
        return Icon(Icons.schedule, size: 14, color: muted);
      case DeliveryStatus.sent:
        return Icon(Icons.check, size: 14, color: muted);
      case DeliveryStatus.delivered:
        return Icon(Icons.done_all, size: 14, color: muted);
      case DeliveryStatus.read:
        return const Icon(Icons.done_all, size: 14, color: AppTheme.read);
      case DeliveryStatus.failed:
        return Icon(Icons.error_outline, size: 14, color: Theme.of(context).colorScheme.error);
    }
  }
}

String _formatTime(DateTime t) {
  final h = t.hour.toString().padLeft(2, '0');
  final m = t.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
