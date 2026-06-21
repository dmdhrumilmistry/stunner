#!/usr/bin/env bash
# Configure the per-platform network permissions Stunner's P2P core needs.
#
# Flutter only grants network access to debug/profile builds by default:
#   - Android: the INTERNET permission lives in the debug/profile manifests, not
#     in src/main, so RELEASE APKs cannot use the network.
#   - macOS: the app is sandboxed; release entitlements grant no network access,
#     and even the debug profile omits the *client* (outbound) entitlement.
# Without these, WebRTC/STUN gathers zero ICE candidates ("No ICE candidates
# gathered") on Android and macOS while Windows (no sandbox) works fine.
#
# This patches the generated platform projects idempotently. It is safe to run
# repeatedly and only touches files that exist, so it is a no-op for platforms
# that have not been generated. Run it after `flutter create`.
#
# Usage: scripts/setup-app-permissions.sh [APP_DIR]   (APP_DIR defaults to "app")
set -euo pipefail

APP_DIR="${1:-app}"

patch_android() {
  local manifest="$APP_DIR/android/app/src/main/AndroidManifest.xml"
  [ -f "$manifest" ] || return 0
  if grep -q 'android.permission.INTERNET' "$manifest"; then
    echo "android: INTERNET permission already present"
    return 0
  fi
  # Insert the permission as the first child of <manifest>.
  awk '
    /<manifest/ && !done {
      print
      print "    <uses-permission android:name=\"android.permission.INTERNET\"/>"
      done = 1
      next
    }
    { print }
  ' "$manifest" > "$manifest.tmp" && mv "$manifest.tmp" "$manifest"
  echo "android: added INTERNET permission to src/main/AndroidManifest.xml"
}

patch_macos() {
  local plistbuddy=/usr/libexec/PlistBuddy
  [ -x "$plistbuddy" ] || { echo "macos: PlistBuddy unavailable, skipping (not macOS)"; return 0; }
  local f key
  for f in "$APP_DIR/macos/Runner/DebugProfile.entitlements" \
           "$APP_DIR/macos/Runner/Release.entitlements"; do
    [ -f "$f" ] || continue
    for key in com.apple.security.network.client com.apple.security.network.server; do
      "$plistbuddy" -c "Add :$key bool true" "$f" 2>/dev/null \
        || "$plistbuddy" -c "Set :$key true" "$f"
    done
    echo "macos: ensured network entitlements in $(basename "$f")"
  done
}

patch_android
patch_macos
echo "App network permissions configured."
