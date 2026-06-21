# Stunner app (Flutter)

The cross-platform UI for Stunner. It calls the Go core (`../core`) over FFI.

## Generating platform projects

This skeleton contains the Dart source (`lib/`) and `pubspec.yaml`. The
per-platform host projects (`android/`, `ios/`, `macos/`, `windows/`) are
generated locally so they aren't checked in as boilerplate:

```bash
cd app
flutter create --platforms=android,ios,macos,windows .
flutter pub get
# Grant the network permissions the P2P core needs (see note below).
bash ../scripts/setup-app-permissions.sh .
```

### Network permissions (required)

Flutter only grants network access to **debug/profile** builds by default, so a
**release** Android APK (no `INTERNET` permission) and the sandboxed **macOS**
app (no network entitlement) gather **zero ICE candidates** — the STUN test
reports *"No ICE candidates gathered"* and P2P cannot connect. Windows is
unaffected. Run `scripts/setup-app-permissions.sh` (or `make app-permissions`
from the repo root) after `flutter create` to patch the generated projects
idempotently; the release CI does this automatically.

## Running

```bash
flutter run            # choose a device / desktop target
flutter analyze
flutter test
```

## Wiring the Go core

- **Desktop:** build the c-shared library and place it where the app can load it
  (see `../docs/ROADMAP.md`):
  ```bash
  cd ../core && go build -buildmode=c-shared -o ../app/native/libstunner.so ./ffi
  ```
  `lib/src/ffi/stunner_ffi.dart` loads `libstunner.{so,dylib,dll}` and degrades
  gracefully if it is missing.
- **Mobile:** bind the core with `gomobile bind` (Android `.aar`, iOS
  `.xcframework`); the mobile FFI path is wired in a later roadmap phase.

The Settings screen ("My safety number" and "Core version") exercises the FFI
boundary as a smoke test.
