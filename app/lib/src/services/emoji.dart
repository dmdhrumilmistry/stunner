/// Emoji helpers: `:shortcode:` expansion (mirrors core/pkg/emoji) and a small
/// built-in picker set (avoids a native plugin dependency).
library;

const _shortcodes = <String, String>{
  'smile': '😄',
  'grin': '😁',
  'joy': '😂',
  'heart': '❤️',
  'thumbsup': '👍',
  'thumbsdown': '👎',
  'fire': '🔥',
  'tada': '🎉',
  'rocket': '🚀',
  'eyes': '👀',
  'wave': '👋',
  'lock': '🔒',
};

/// Replaces `:shortcode:` tokens with their Unicode emoji. Unknown shortcodes
/// are left untouched.
String expandShortcodes(String input) {
  return input.replaceAllMapped(RegExp(r':([a-z0-9_]+):'), (m) {
    return _shortcodes[m.group(1)] ?? m.group(0)!;
  });
}

/// A compact set of emoji for the in-app picker.
const pickerEmojis = <String>[
  '😀', '😄', '😁', '😂', '🙂', '😉', '😍', '😘',
  '😎', '🤔', '😴', '😢', '😭', '😡', '🥳', '🤝',
  '👍', '👎', '👏', '🙏', '👋', '💪', '🔥', '🎉',
  '✨', '⭐', '❤️', '🧡', '💜', '💯', '🚀', '📎',
  '🔒', '🗝️', '📝', '📷', '🎵', '☕', '🍕', '🌙',
];
