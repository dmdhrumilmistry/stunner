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
```

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
