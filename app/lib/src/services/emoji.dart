/// Emoji shortcode expansion, mirroring core/pkg/emoji so message text can be
/// composed with `:shortcode:` tokens. Static emoji render natively; animated
/// emoji (Lottie/APNG/WebP) are referenced by id and resolved separately.
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
