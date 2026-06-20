import 'package:flutter/material.dart';

/// Stunner's visual theme, mirroring the shadcn/ui "neutral" design system used
/// in the product design (grayscale surfaces, near-black/near-white primary, a
/// single green accent for presence). OKLCH tokens from the design are converted
/// to their sRGB equivalents here.
class AppTheme {
  AppTheme._();

  /// Presence / success accent (oklch(0.72 0.17 152)).
  static const online = Color(0xFF22C55E);

  /// Read-receipt "read" tint.
  static const read = Color(0xFF3B82F6);

  // --- neutral ramp (shadcn base "neutral") ---
  static const _white = Color(0xFFFFFFFF);
  static const _n50 = Color(0xFFFAFAFA); // oklch 0.985
  static const _n100 = Color(0xFFF5F5F5); // oklch 0.97
  static const _n200 = Color(0xFFE5E5E5); // oklch 0.922 (border)
  static const _n400 = Color(0xFFA1A1A1); // oklch 0.708 (ring)
  static const _n500 = Color(0xFF8A8A8A); // oklch 0.556 (muted fg)
  static const _n700 = Color(0xFF404040); // oklch 0.269 (dark secondary)
  static const _n800 = Color(0xFF333333); // oklch 0.205 (primary / dark card)
  static const _n900 = Color(0xFF252525); // oklch 0.145 (fg / dark bg)

  static const _redLight = Color(0xFFDC2626);
  static const _redDark = Color(0xFFF05252);

  static ThemeData light() => _build(Brightness.light);
  static ThemeData dark() => _build(Brightness.dark);

  static ThemeData _build(Brightness brightness) {
    final isDark = brightness == Brightness.dark;

    final scheme = isDark
        ? const ColorScheme.dark(
            primary: _n50,
            onPrimary: _n800,
            secondary: _n700,
            onSecondary: _n50,
            surface: _n900,
            onSurface: _n50,
            surfaceContainerLowest: _n900,
            surfaceContainerLow: _n800,
            surfaceContainer: _n800,
            surfaceContainerHigh: _n700,
            surfaceContainerHighest: _n700,
            onSurfaceVariant: Color(0xFFB5B5B5),
            outline: Color(0xFF3A3A3A),
            outlineVariant: Color(0xFF2F2F2F),
            error: _redDark,
            onError: _n50,
          )
        : const ColorScheme.light(
            primary: _n800,
            onPrimary: _n50,
            secondary: _n100,
            onSecondary: _n800,
            surface: _white,
            onSurface: _n900,
            surfaceContainerLowest: _white,
            surfaceContainerLow: _n50,
            surfaceContainer: _n100,
            surfaceContainerHigh: _n100,
            surfaceContainerHighest: _n100,
            onSurfaceVariant: _n500,
            outline: _n200,
            outlineVariant: _n200,
            error: _redLight,
            onError: _white,
          );

    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: scheme,
      scaffoldBackgroundColor: scheme.surface,
      dividerColor: scheme.outline,
      splashFactory: InkSparkle.splashFactory,
      appBarTheme: AppBarTheme(
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0.5,
        centerTitle: false,
        titleTextStyle: TextStyle(
          color: scheme.onSurface,
          fontSize: 18,
          fontWeight: FontWeight.w700,
          letterSpacing: -0.3,
        ),
      ),
      dividerTheme: DividerThemeData(color: scheme.outline, thickness: 1, space: 1),
      cardTheme: CardThemeData(
        color: scheme.surfaceContainer,
        elevation: 0,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
          textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
          side: BorderSide(color: scheme.outline),
          foregroundColor: scheme.onSurface,
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: scheme.surfaceContainer,
        contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        hintStyle: TextStyle(color: scheme.onSurfaceVariant),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(11),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(11),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(11),
          borderSide: const BorderSide(color: _n400, width: 1.5),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: scheme.surface,
        indicatorColor: scheme.surfaceContainerHigh,
        elevation: 0,
        labelTextStyle: WidgetStateProperty.all(
          const TextStyle(fontSize: 11, fontWeight: FontWeight.w500),
        ),
      ),
      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: scheme.surfaceContainerLow,
        indicatorColor: scheme.surfaceContainerHigh,
        selectedIconTheme: IconThemeData(color: scheme.onSurface),
        unselectedIconTheme: IconThemeData(color: scheme.onSurfaceVariant),
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.selected) ? scheme.onPrimary : scheme.onSurfaceVariant,
        ),
        trackColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.selected) ? scheme.primary : scheme.surfaceContainerHighest,
        ),
        trackOutlineColor: WidgetStateProperty.all(Colors.transparent),
      ),
      listTileTheme: const ListTileThemeData(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(12))),
      ),
    );
  }
}
