import 'dart:async';
import 'dart:convert';

import '../ffi/stunner_ffi.dart';

/// A decoded asynchronous notification from the Go core. Mirrors the runtime's
/// `session.Event` JSON (see core/pkg/session). The FFI delivers these by
/// polling; this abstraction hides that so the rest of the app just listens to
/// a [Stream].
sealed class CoreEvent {
  const CoreEvent();

  /// Decodes one event map from [StunnerCore.pollEventsJson]. Returns null for
  /// kinds the app does not handle.
  static CoreEvent? fromJson(Map<String, dynamic> m) {
    switch (m['kind'] as String?) {
      case 'message':
        return IncomingMessage(
          convId: m['convId'] as String? ?? '',
          peerFp: m['peerFp'] as String? ?? '',
          text: m['text'] as String? ?? '',
          msgId: m['msgId'] as String? ?? '',
        );
      case 'connected':
        return PeerConnected(peerFp: m['peerFp'] as String? ?? '');
      case 'disconnected':
        return PeerDisconnected(peerFp: m['peerFp'] as String? ?? '');
      case 'error':
        return CoreError(message: m['err'] as String? ?? 'unknown error');
      default:
        return null;
    }
  }
}

class IncomingMessage extends CoreEvent {
  const IncomingMessage({
    required this.convId,
    required this.peerFp,
    required this.text,
    required this.msgId,
  });

  final String convId;
  final String peerFp;
  final String text;
  final String msgId;
}

class PeerConnected extends CoreEvent {
  const PeerConnected({required this.peerFp});
  final String peerFp;
}

class PeerDisconnected extends CoreEvent {
  const PeerDisconnected({required this.peerFp});
  final String peerFp;
}

class CoreError extends CoreEvent {
  const CoreError({required this.message});
  final String message;
}

/// A unified source of [CoreEvent]s. Today only [PollingEventSource] exists; a
/// callback-based source (gomobile EventHandler bridged via a platform channel)
/// could be dropped in behind this interface with no downstream changes.
abstract class CoreEventSource {
  Stream<CoreEvent> get events;
  void start();
  void pause();
  void resume();
  void dispose();
}

/// Polls [StunnerCore.pollEventsJson] on a timer and emits decoded events. The
/// poll is a non-blocking buffer drain in the core (microseconds), so running
/// it on the UI isolate via a [Timer] is fine for text messaging.
class PollingEventSource implements CoreEventSource {
  PollingEventSource(
    this._core, {
    this.interval = const Duration(milliseconds: 150),
  });

  final StunnerCore _core;
  final Duration interval;
  final _controller = StreamController<CoreEvent>.broadcast();
  Timer? _timer;

  @override
  Stream<CoreEvent> get events => _controller.stream;

  @override
  void start() {
    if (!_core.available || _timer != null) return;
    _timer = Timer.periodic(interval, (_) => _tick());
  }

  void _tick() {
    // Never let a transient failure kill the timer.
    try {
      final raw = _core.pollEventsJson();
      final decoded = jsonDecode(raw);
      if (decoded is! List) return;
      for (final item in decoded) {
        if (item is Map<String, dynamic>) {
          final ev = CoreEvent.fromJson(item);
          if (ev != null) _controller.add(ev);
        }
      }
    } catch (e) {
      _controller.add(CoreError(message: 'poll failed: $e'));
    }
  }

  @override
  void pause() {
    _timer?.cancel();
    _timer = null;
  }

  @override
  void resume() => start();

  @override
  void dispose() {
    pause();
    _controller.close();
  }
}
