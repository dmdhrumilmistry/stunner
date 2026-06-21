import 'package:flutter/material.dart';

import '../ffi/stunner_ffi.dart';
import '../models/chat.dart';
import '../services/app_state.dart';
import '../services/chat_store.dart';
import '../services/notification_service.dart';
import 'contact_profile_view.dart';
import 'conversation_view.dart';
import 'settings_views.dart';
import 'widgets.dart';

enum HomeTab { chats, contacts, settings }

/// The app's responsive home. On wide screens it is a three-column layout
/// (nav rail · list · detail); on narrow screens it is a single list pane with a
/// bottom tab bar, where opening an item pushes a full-screen detail route.
class HomeShell extends StatefulWidget {
  const HomeShell({
    super.key,
    required this.core,
    required this.store,
    required this.appState,
    required this.notifications,
  });

  final StunnerCore core;
  final ChatStore store;
  final AppState appState;
  final NotificationService notifications;

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  HomeTab _tab = HomeTab.chats;
  String? _selectedChatId;
  String? _selectedContactId;
  String _settingsSection = 'profile';
  String _search = '';
  bool _wide = false;
  int _lastNotifVersion = 0;
  String? _narrowOpenChatId;

  ChatStore get store => widget.store;
  AppState get appState => widget.appState;
  StunnerCore get core => widget.core;
  NotificationService get notifications => widget.notifications;

  @override
  void initState() {
    super.initState();
    _lastNotifVersion = notifications.version;
    notifications.addListener(_onNotification);
  }

  @override
  void dispose() {
    notifications.removeListener(_onNotification);
    super.dispose();
  }

  /// Shows a one-shot in-app banner when a new notification arrives for a chat
  /// the user is not currently viewing.
  void _onNotification() {
    if (notifications.version == _lastNotifVersion) return;
    _lastNotifVersion = notifications.version;
    final latest = notifications.latest;
    if (latest == null || !mounted) return;
    if (_isChatVisible(latest.chatId)) return;
    final messenger = ScaffoldMessenger.of(context);
    messenger.clearSnackBars();
    messenger.showSnackBar(SnackBar(
      behavior: SnackBarBehavior.floating,
      content: Text('${latest.title}: ${latest.body}',
          maxLines: 2, overflow: TextOverflow.ellipsis),
      action: SnackBarAction(label: 'View', onPressed: () => _openChat(latest.chatId)),
    ));
  }

  bool _isChatVisible(String chatId) =>
      (_wide && _tab == HomeTab.chats && _selectedChatId == chatId) ||
      (!_wide && _narrowOpenChatId == chatId);

  // --- navigation ---

  void _selectTab(HomeTab tab) {
    setState(() {
      _tab = tab;
      _search = '';
    });
  }

  void _openChat(String chatId) {
    store.markRead(chatId);
    if (_wide) {
      setState(() {
        _tab = HomeTab.chats;
        _selectedChatId = chatId;
      });
    } else {
      _narrowOpenChatId = chatId;
      Navigator.of(context)
          .push(MaterialPageRoute<void>(
            builder: (ctx) => Scaffold(
              body: SafeArea(
                child: ConversationView(
                  store: store,
                  appState: appState,
                  chatId: chatId,
                  showBack: true,
                  onBack: () => Navigator.of(ctx).pop(),
                  onOpenContact: _openContact,
                ),
              ),
            ),
          ))
          .then((_) {
        if (_narrowOpenChatId == chatId) _narrowOpenChatId = null;
      });
    }
  }

  /// Opens the notification center (recent in-app notifications).
  Future<void> _openNotifications() async {
    final chatId = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) => ListenableBuilder(
        listenable: notifications,
        builder: (ctx, _) {
          final items = notifications.items;
          return SafeArea(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 8, 0),
                  child: Row(
                    children: [
                      const Expanded(
                        child: Text('Notifications',
                            style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700)),
                      ),
                      if (items.isNotEmpty) ...[
                        TextButton(
                          onPressed: notifications.markAllRead,
                          child: const Text('Mark read'),
                        ),
                        TextButton(
                          onPressed: notifications.clear,
                          child: const Text('Clear'),
                        ),
                      ],
                    ],
                  ),
                ),
                if (items.isEmpty)
                  const Padding(
                    padding: EdgeInsets.all(28),
                    child: Text('No notifications yet'),
                  ),
                Flexible(
                  child: ListView.builder(
                    shrinkWrap: true,
                    itemCount: items.length,
                    itemBuilder: (ctx, i) {
                      final n = items[i];
                      return ListTile(
                        leading: CircleAvatar(
                          backgroundColor: Theme.of(ctx).colorScheme.surfaceContainerHighest,
                          child: Icon(
                            n.read ? Icons.notifications_none : Icons.notifications_active,
                            size: 20,
                            color: Theme.of(ctx).colorScheme.onSurfaceVariant,
                          ),
                        ),
                        title: Text(n.title, style: const TextStyle(fontWeight: FontWeight.w600)),
                        subtitle: Text(n.body, maxLines: 1, overflow: TextOverflow.ellipsis),
                        onTap: () {
                          notifications.markRead(n.id);
                          Navigator.pop(ctx, n.chatId);
                        },
                      );
                    },
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
    if (chatId != null && store.maybeChat(chatId) != null) {
      _openChat(chatId);
    }
  }

  void _openContact(String contactId) {
    if (_wide) {
      setState(() {
        _tab = HomeTab.contacts;
        _selectedContactId = contactId;
      });
    } else {
      Navigator.of(context).push(MaterialPageRoute<void>(
        builder: (ctx) => Scaffold(
          body: SafeArea(
            child: ContactProfileView(
              store: store,
              contactId: contactId,
              showBack: true,
              onBack: () => Navigator.of(ctx).pop(),
              onMessage: _messageContact,
              onDeleted: () => Navigator.of(ctx).pop(),
            ),
          ),
        ),
      ));
    }
  }

  void _messageContact(String contactId) {
    final c = store.contactById(contactId);
    if (c == null) return;
    final id = store.startChatWith(c);
    if (_wide) {
      setState(() {
        _tab = HomeTab.chats;
        _selectedChatId = id;
      });
    } else {
      Navigator.of(context).popUntil((r) => r.isFirst);
      _openChat(id);
    }
  }

  void _openSection(String section) {
    if (_wide) {
      setState(() => _settingsSection = section);
    } else {
      Navigator.of(context).push(MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          appBar: AppBar(title: Text(_sectionTitle(section))),
          body: SafeArea(child: _sectionBody(section)),
        ),
      ));
    }
  }

  // --- compose / add contact ---

  Future<void> _startChat() async {
    const addSentinel = '__add__';
    final choice = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) {
        final contacts = store.contacts;
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.person_add_alt),
                title: const Text('Add new contact'),
                onTap: () => Navigator.pop(ctx, addSentinel),
              ),
              const Divider(height: 1),
              if (contacts.isEmpty)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Text('No contacts yet — add one to start chatting.'),
                ),
              for (final c in contacts)
                ListTile(
                  leading: Avatar(initials: c.initials, size: 40, online: c.online, showDot: true),
                  title: Text(c.name),
                  subtitle: c.role.isEmpty ? null : Text(c.role),
                  onTap: () => Navigator.pop(ctx, c.id),
                ),
            ],
          ),
        );
      },
    );
    if (choice == null || !mounted) return;
    if (choice == addSentinel) {
      final contact = await _addContact();
      if (contact != null) _openChat(store.startChatWith(contact));
      return;
    }
    final c = store.contactById(choice);
    if (c != null) _openChat(store.startChatWith(c));
  }

  Future<Contact?> _addContact() async {
    final nameCtl = TextEditingController();
    final codeCtl = TextEditingController();
    String? error;
    return showDialog<Contact>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: const Text('Add contact'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: codeCtl,
                autofocus: true,
                decoration: const InputDecoration(
                  labelText: 'Contact ID (required)',
                  hintText: 'stunner:contact?...',
                  helperText: 'Paste the code they shared from My identity.',
                ),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: nameCtl,
                decoration: const InputDecoration(labelText: 'Name (optional)'),
              ),
              if (error != null) ...[
                const SizedBox(height: 8),
                Text(error!, style: TextStyle(color: Theme.of(ctx).colorScheme.error)),
              ],
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
            FilledButton(
              onPressed: () {
                final code = codeCtl.text.trim();
                var name = nameCtl.text.trim();
                if (code.isEmpty) {
                  setLocal(() => error = 'Enter the contact ID (their stunner:contact code).');
                  return;
                }
                if (!core.available) {
                  setLocal(() => error = 'Core library not loaded; cannot add contacts.');
                  return;
                }
                late final String fingerprint;
                try {
                  final info = core.validateContactURI(code);
                  fingerprint = info.fingerprint;
                  if (name.isEmpty) name = info.handle.isEmpty ? 'Contact' : info.handle;
                } on FormatException catch (e) {
                  setLocal(() => error = 'Invalid contact ID: ${e.message}');
                  return;
                }
                Navigator.pop(ctx, store.addContact(name: name, code: code, fingerprint: fingerprint));
              },
              child: const Text('Add'),
            ),
          ],
        ),
      ),
    );
  }

  // --- build ---

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        _wide = constraints.maxWidth >= 900;
        if (_wide) return _wideLayout();
        return _narrowLayout();
      },
    );
  }

  Widget _wideLayout() {
    return Scaffold(
      body: SafeArea(
        child: Row(
          children: [
            _navRail(),
            VerticalDivider(width: 1, color: Theme.of(context).colorScheme.outline),
            SizedBox(width: 340, child: _listPane()),
            VerticalDivider(width: 1, color: Theme.of(context).colorScheme.outline),
            Expanded(child: _detailPane()),
          ],
        ),
      ),
    );
  }

  Widget _narrowLayout() {
    return Scaffold(
      body: SafeArea(bottom: false, child: _listPane()),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab.index,
        onDestinationSelected: (i) => _selectTab(HomeTab.values[i]),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.chat_bubble_outline), selectedIcon: Icon(Icons.chat_bubble), label: 'Chats'),
          NavigationDestination(icon: Icon(Icons.people_outline), selectedIcon: Icon(Icons.people), label: 'Contacts'),
          NavigationDestination(icon: Icon(Icons.settings_outlined), selectedIcon: Icon(Icons.settings), label: 'Settings'),
        ],
      ),
      floatingActionButton: _tab == HomeTab.chats
          ? FloatingActionButton(
              onPressed: _startChat,
              tooltip: 'New chat',
              child: const Icon(Icons.edit_outlined),
            )
          : _tab == HomeTab.contacts
              ? FloatingActionButton(
                  onPressed: () async {
                    final c = await _addContact();
                    if (c != null) _openContact(c.id);
                  },
                  tooltip: 'Add contact',
                  child: const Icon(Icons.person_add_alt),
                )
              : null,
    );
  }

  Widget _navRail() {
    final scheme = Theme.of(context).colorScheme;
    Widget railBtn(IconData icon, HomeTab tab, String tooltip) {
      final selected = _tab == tab;
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: IconButton(
          tooltip: tooltip,
          isSelected: selected,
          onPressed: () => _selectTab(tab),
          style: IconButton.styleFrom(
            backgroundColor: selected ? scheme.surfaceContainerHigh : Colors.transparent,
            foregroundColor: selected ? scheme.onSurface : scheme.onSurfaceVariant,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            padding: const EdgeInsets.all(12),
          ),
          icon: Icon(icon),
        ),
      );
    }

    return Container(
      width: 72,
      color: scheme.surfaceContainerLow,
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Column(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: scheme.primary,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(Icons.chat_bubble, size: 20, color: scheme.onPrimary),
          ),
          const SizedBox(height: 14),
          railBtn(Icons.chat_bubble_outline, HomeTab.chats, 'Chats'),
          railBtn(Icons.people_outline, HomeTab.contacts, 'Contacts'),
          railBtn(Icons.settings_outlined, HomeTab.settings, 'Settings'),
          const Spacer(),
          ListenableBuilder(
            listenable: appState,
            builder: (context, _) => IconButton(
              tooltip: 'Toggle theme',
              onPressed: () => appState.toggleTheme(context),
              icon: Icon(appState.isDark(context) ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
            ),
          ),
          const SizedBox(height: 6),
          ListenableBuilder(
            listenable: appState,
            builder: (context, _) => InkWell(
              borderRadius: BorderRadius.circular(99),
              onTap: () {
                setState(() {
                  _tab = HomeTab.settings;
                  _settingsSection = 'profile';
                });
              },
              child: Avatar(initials: appState.profile.initials, size: 44, online: true, showDot: true),
            ),
          ),
        ],
      ),
    );
  }

  Widget _listPane() {
    switch (_tab) {
      case HomeTab.chats:
        return _chatsList();
      case HomeTab.contacts:
        return _contactsList();
      case HomeTab.settings:
        return _settingsList();
    }
  }

  Widget _detailPane() {
    switch (_tab) {
      case HomeTab.chats:
        if (_selectedChatId != null && store.maybeChat(_selectedChatId!) != null) {
          return ConversationView(
            key: ValueKey(_selectedChatId),
            store: store,
            appState: appState,
            chatId: _selectedChatId!,
            onOpenContact: _openContact,
          );
        }
        return _emptyDetail(Icons.chat_bubble_outline, 'No conversation selected', 'Choose a chat to start messaging');
      case HomeTab.contacts:
        if (_selectedContactId != null && store.contactById(_selectedContactId!) != null) {
          return ContactProfileView(
            key: ValueKey(_selectedContactId),
            store: store,
            contactId: _selectedContactId!,
            onMessage: _messageContact,
            onDeleted: () => setState(() => _selectedContactId = null),
          );
        }
        return _emptyDetail(Icons.people_outline, 'No contact selected', 'Choose someone to see their profile');
      case HomeTab.settings:
        return Column(
          children: [
            _detailHeader(_sectionTitle(_settingsSection)),
            Divider(height: 1, color: Theme.of(context).colorScheme.outline),
            Expanded(child: _sectionBody(_settingsSection)),
          ],
        );
    }
  }

  Widget _detailHeader(String title) {
    return Container(
      height: 56,
      alignment: Alignment.centerLeft,
      padding: const EdgeInsets.symmetric(horizontal: 18),
      child: Text(title, style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w600)),
    );
  }

  Widget _emptyDetail(IconData icon, String title, String sub) {
    final scheme = Theme.of(context).colorScheme;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 72,
            height: 72,
            decoration: BoxDecoration(color: scheme.surfaceContainer, shape: BoxShape.circle),
            child: Icon(icon, size: 30, color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 14),
          Text(title, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          Text(sub, style: TextStyle(fontSize: 13.5, color: scheme.onSurfaceVariant)),
        ],
      ),
    );
  }

  // --- list panes ---

  Widget _paneHeader(String title, {List<Widget> actions = const []}) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 14, 10, 10),
      child: Row(
        children: [
          Expanded(
            child: Text(title,
                style: const TextStyle(fontSize: 26, fontWeight: FontWeight.w700, letterSpacing: -0.6)),
          ),
          ...actions,
        ],
      ),
    );
  }

  Widget _searchField(String hint) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: TextField(
        onChanged: (v) => setState(() => _search = v),
        decoration: InputDecoration(
          hintText: hint,
          prefixIcon: const Icon(Icons.search, size: 20),
        ),
      ),
    );
  }

  Widget _chatsList() {
    return ListenableBuilder(
      listenable: store,
      builder: (context, _) {
        final q = _search.toLowerCase();
        final chats = store.chats.where((c) {
          if (q.isEmpty) return true;
          return c.name.toLowerCase().contains(q) ||
              (c.last?.text.toLowerCase().contains(q) ?? false);
        }).toList();
        return Column(
          children: [
            _paneHeader('Chats', actions: [
              ListenableBuilder(
                listenable: notifications,
                builder: (context, _) {
                  final count = notifications.unreadCount;
                  return IconButton(
                    tooltip: 'Notifications',
                    onPressed: _openNotifications,
                    icon: Badge(
                      isLabelVisible: count > 0,
                      label: Text('$count'),
                      child: const Icon(Icons.notifications_outlined),
                    ),
                  );
                },
              ),
              ListenableBuilder(
                listenable: appState,
                builder: (context, _) => IconButton(
                  tooltip: 'Toggle theme',
                  onPressed: () => appState.toggleTheme(context),
                  icon: Icon(appState.isDark(context)
                      ? Icons.light_mode_outlined
                      : Icons.dark_mode_outlined),
                ),
              ),
              IconButton.filled(
                tooltip: 'New chat',
                onPressed: _startChat,
                icon: const Icon(Icons.edit_outlined, size: 20),
              ),
            ]),
            _searchField('Search'),
            Expanded(
              child: chats.isEmpty
                  ? _listEmpty('No conversations yet', 'Tap the compose button to start chatting.')
                  : ListView.builder(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      itemCount: chats.length,
                      itemBuilder: (context, i) => _chatRow(chats[i]),
                    ),
            ),
          ],
        );
      },
    );
  }

  Widget _chatRow(Chat chat) {
    final scheme = Theme.of(context).colorScheme;
    final contact = store.contactForChat(chat);
    final last = chat.last;
    final selected = _wide && _selectedChatId == chat.id;
    return Dismissible(
      key: ValueKey(chat.id),
      direction: DismissDirection.endToStart,
      background: Container(
        decoration: BoxDecoration(
          color: scheme.errorContainer,
          borderRadius: BorderRadius.circular(14),
        ),
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.symmetric(horizontal: 20),
        child: Icon(Icons.delete_outline, color: scheme.onErrorContainer),
      ),
      confirmDismiss: (_) => _confirmDelete('Delete chat with ${chat.name}?',
          'This removes the conversation from this device.'),
      onDismissed: (_) {
        store.deleteChat(chat.id);
        if (_selectedChatId == chat.id) setState(() => _selectedChatId = null);
      },
      child: Material(
        color: selected ? scheme.surfaceContainerHigh : Colors.transparent,
        borderRadius: BorderRadius.circular(14),
        child: InkWell(
          borderRadius: BorderRadius.circular(14),
          onTap: () => _openChat(chat.id),
          onLongPress: () => _chatMenu(chat),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 9),
            child: Row(
              children: [
                Avatar(
                  initials: contact?.initials ?? '?',
                  size: 52,
                  online: contact?.online ?? false,
                  showDot: true,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(chat.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                          ),
                          if (last != null)
                            Text(_formatTime(last.time),
                                style: TextStyle(
                                    fontSize: 12,
                                    color: chat.unread > 0 ? scheme.primary : scheme.onSurfaceVariant)),
                        ],
                      ),
                      const SizedBox(height: 3),
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              last?.text ?? 'No messages yet',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(fontSize: 13.5, color: scheme.onSurfaceVariant),
                            ),
                          ),
                          if (chat.unread > 0)
                            Container(
                              margin: const EdgeInsets.only(left: 6),
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                              constraints: const BoxConstraints(minWidth: 20),
                              decoration: BoxDecoration(
                                  color: scheme.primary, borderRadius: BorderRadius.circular(99)),
                              child: Text('${chat.unread}',
                                  textAlign: TextAlign.center,
                                  style: TextStyle(
                                      fontSize: 11.5,
                                      fontWeight: FontWeight.w700,
                                      color: scheme.onPrimary)),
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _contactsList() {
    return ListenableBuilder(
      listenable: store,
      builder: (context, _) {
        final q = _search.toLowerCase();
        final contacts = store.contacts.where((c) {
          if (q.isEmpty) return true;
          return c.name.toLowerCase().contains(q) || c.role.toLowerCase().contains(q);
        }).toList()
          ..sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
        return Column(
          children: [
            _paneHeader('Contacts', actions: [
              IconButton.filled(
                tooltip: 'Add contact',
                onPressed: () async {
                  final c = await _addContact();
                  if (c != null) _openContact(c.id);
                },
                icon: const Icon(Icons.person_add_alt, size: 20),
              ),
            ]),
            _searchField('Search contacts'),
            Expanded(
              child: contacts.isEmpty
                  ? _listEmpty('No contacts yet', 'Add a contact to start chatting.')
                  : ListView.builder(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      itemCount: contacts.length,
                      itemBuilder: (context, i) => _contactRow(contacts[i]),
                    ),
            ),
          ],
        );
      },
    );
  }

  Widget _contactRow(Contact c) {
    final scheme = Theme.of(context).colorScheme;
    final selected = _wide && _selectedContactId == c.id;
    return Material(
      color: selected ? scheme.surfaceContainerHigh : Colors.transparent,
      borderRadius: BorderRadius.circular(14),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: () => _openContact(c.id),
        onLongPress: () => _contactMenu(c),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 9),
          child: Row(
            children: [
              Avatar(initials: c.initials, size: 44, online: c.online, showDot: true),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(c.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 15.5, fontWeight: FontWeight.w600)),
                    if (c.role.isNotEmpty)
                      Text(c.role,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(fontSize: 13, color: scheme.onSurfaceVariant)),
                  ],
                ),
              ),
              Icon(Icons.chevron_right, color: scheme.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }

  Widget _settingsList() {
    final scheme = Theme.of(context).colorScheme;
    Widget row(IconData icon, String label, String section) {
      final selected = _wide && _settingsSection == section;
      return InkWell(
        onTap: () => _openSection(section),
        child: Container(
          color: selected ? scheme.surfaceContainerHigh : Colors.transparent,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
          child: Row(
            children: [
              Container(
                width: 30,
                height: 30,
                decoration: BoxDecoration(color: scheme.surface, borderRadius: BorderRadius.circular(8)),
                child: Icon(icon, size: 17),
              ),
              const SizedBox(width: 14),
              Expanded(child: Text(label, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w500))),
              Icon(Icons.chevron_right, size: 18, color: scheme.onSurfaceVariant),
            ],
          ),
        ),
      );
    }

    return ListenableBuilder(
      listenable: appState,
      builder: (context, _) => Column(
        children: [
          _paneHeader('Settings'),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(16, 6, 16, 16),
              children: [
                InkWell(
                  borderRadius: BorderRadius.circular(16),
                  onTap: () => _openSection('profile'),
                  child: Container(
                    padding: const EdgeInsets.all(14),
                    decoration: BoxDecoration(
                        color: scheme.surfaceContainer, borderRadius: BorderRadius.circular(16)),
                    child: Row(
                      children: [
                        Avatar(initials: appState.profile.initials, size: 52),
                        const SizedBox(width: 14),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(appState.profile.name,
                                  style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w600)),
                              const SizedBox(height: 2),
                              Text(appState.profile.status,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(fontSize: 13.5, color: scheme.onSurfaceVariant)),
                            ],
                          ),
                        ),
                        Icon(Icons.chevron_right, color: scheme.onSurfaceVariant),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 18),
                SettingsCard(children: [
                  row(Icons.person_outline, 'Profile', 'profile'),
                  row(Icons.brightness_6_outlined, 'Appearance', 'appearance'),
                  row(Icons.notifications_outlined, 'Notifications', 'notifications'),
                  row(Icons.wifi_tethering, 'Network', 'network'),
                  row(Icons.shield_outlined, 'Privacy & Safety', 'privacy'),
                ]),
                const SizedBox(height: 18),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton(
                    onPressed: _logout,
                    child: const Text('Log out'),
                  ),
                ),
                const SizedBox(height: 18),
                Center(
                  child: Text(
                    'Stunner · Android · iOS · macOS · Windows\n${core.version()}',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 12, color: scheme.onSurfaceVariant),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // --- settings sections ---

  String _sectionTitle(String section) {
    switch (section) {
      case 'appearance':
        return 'Appearance';
      case 'notifications':
        return 'Notifications';
      case 'network':
        return 'Network';
      case 'privacy':
        return 'Privacy & Safety';
      case 'profile':
      default:
        return 'Profile';
    }
  }

  Widget _sectionBody(String section) {
    switch (section) {
      case 'appearance':
        return AppearanceView(appState: appState);
      case 'notifications':
        return NotificationsView(appState: appState);
      case 'network':
        return NetworkView(core: core);
      case 'privacy':
        return PrivacyView(appState: appState, core: core);
      case 'profile':
      default:
        return ProfileEditView(appState: appState, core: core);
    }
  }

  // --- helpers ---

  Future<void> _logout() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Log out?'),
        content: const Text('Your encrypted identity stays on this device.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Log out')),
        ],
      ),
    );
    if (ok == true && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Sign-in/out needs the persistent account (coming soon).')),
      );
    }
  }

  Future<void> _chatMenu(Chat chat) async {
    final action = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.chat_bubble_outline),
              title: const Text('Open chat'),
              onTap: () => Navigator.pop(ctx, 'open'),
            ),
            ListTile(
              leading: Icon(Icons.delete_outline, color: Theme.of(ctx).colorScheme.error),
              title: Text('Delete chat',
                  style: TextStyle(color: Theme.of(ctx).colorScheme.error)),
              onTap: () => Navigator.pop(ctx, 'delete'),
            ),
          ],
        ),
      ),
    );
    if (action == 'open') {
      _openChat(chat.id);
    } else if (action == 'delete') {
      final ok = await _confirmDelete(
          'Delete chat with ${chat.name}?', 'This removes the conversation from this device.');
      if (ok) {
        store.deleteChat(chat.id);
        if (_selectedChatId == chat.id) setState(() => _selectedChatId = null);
      }
    }
  }

  Future<void> _contactMenu(Contact c) async {
    final action = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.chat_bubble_outline),
              title: const Text('Message'),
              onTap: () => Navigator.pop(ctx, 'message'),
            ),
            ListTile(
              leading: const Icon(Icons.person_outline),
              title: const Text('View profile'),
              onTap: () => Navigator.pop(ctx, 'profile'),
            ),
            ListTile(
              leading: Icon(Icons.delete_outline, color: Theme.of(ctx).colorScheme.error),
              title: Text('Delete contact',
                  style: TextStyle(color: Theme.of(ctx).colorScheme.error)),
              onTap: () => Navigator.pop(ctx, 'delete'),
            ),
          ],
        ),
      ),
    );
    if (action == 'message') {
      _messageContact(c.id);
    } else if (action == 'profile') {
      _openContact(c.id);
    } else if (action == 'delete') {
      final ok = await _confirmDelete('Delete ${c.name}?', 'This removes the contact from this device.');
      if (ok) {
        store.deleteContact(c.id);
        if (_selectedContactId == c.id) setState(() => _selectedContactId = null);
      }
    }
  }

  Future<bool> _confirmDelete(String title, String body) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(body),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Delete')),
        ],
      ),
    );
    return ok ?? false;
  }

  Widget _listEmpty(String title, String sub) {
    final scheme = Theme.of(context).colorScheme;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.chat_bubble_outline, size: 56, color: scheme.onSurfaceVariant),
          const SizedBox(height: 12),
          Text(title, style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          Text(sub, textAlign: TextAlign.center, style: TextStyle(color: scheme.onSurfaceVariant)),
        ],
      ),
    );
  }
}

String _formatTime(DateTime t) {
  final h = t.hour.toString().padLeft(2, '0');
  final m = t.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
