import 'package:flutter/widgets.dart';

import 'krayn_localizations.dart';

class LanguageOption {
  const LanguageOption({
    required this.locale,
    required this.label,
  });

  final Locale locale;
  final String label;

  String get tag => locale.toLanguageTag();
}

class LanguageCatalog {
  const LanguageCatalog._();

  static final options = List<LanguageOption>.unmodifiable(
    KrayNLocalizations.supportedLocales.map(
      (locale) => LanguageOption(
        locale: locale,
        label: _labelFor(locale),
      ),
    ),
  );

  static String _labelFor(Locale locale) {
    final language = _languageNames[locale.languageCode] ?? locale.languageCode;
    final script = locale.scriptCode;
    final country = locale.countryCode;
    final qualifiers = [
      if (script != null && script.isNotEmpty) script,
      if (country != null && country.isNotEmpty) country,
    ];
    if (qualifiers.isEmpty) {
      return language;
    }
    return '$language (${qualifiers.join('-')})';
  }
}

const _languageNames = {
  'af': 'Afrikaans',
  'am': 'Amharic',
  'ar': 'العربية',
  'as': 'Assamese',
  'az': 'Azərbaycanca',
  'be': 'Беларуская',
  'bg': 'Български',
  'bn': 'বাংলা',
  'bo': 'Tibetan',
  'bs': 'Bosanski',
  'ca': 'Catala',
  'cs': 'Cestina',
  'cy': 'Cymraeg',
  'da': 'Dansk',
  'de': 'Deutsch',
  'el': 'Ελληνικά',
  'en': 'English',
  'es': 'Espanol',
  'et': 'Eesti',
  'eu': 'Euskara',
  'fa': 'فارسی',
  'fi': 'Suomi',
  'fil': 'Filipino',
  'fr': 'Francais',
  'ga': 'Gaeilge',
  'gl': 'Galego',
  'gsw': 'Schwiizertüütsch',
  'gu': 'Gujarati',
  'he': 'עברית',
  'hi': 'Hindi',
  'hr': 'Hrvatski',
  'hu': 'Magyar',
  'hy': 'Հայերեն',
  'id': 'Indonesia',
  'is': 'Islenska',
  'it': 'Italiano',
  'ja': '日本語',
  'ka': 'ქართული',
  'kk': 'Қазақша',
  'km': 'Khmer',
  'kn': 'Kannada',
  'ko': '한국어',
  'ky': 'Кыргызча',
  'lo': 'Lao',
  'lt': 'Lietuviu',
  'lv': 'Latviesu',
  'mk': 'Македонски',
  'ml': 'Malayalam',
  'mn': 'Монгол',
  'mr': 'Marathi',
  'ms': 'Melayu',
  'my': 'Burmese',
  'nb': 'Norsk bokmal',
  'ne': 'Nepali',
  'nl': 'Nederlands',
  'no': 'Norsk',
  'or': 'Odia',
  'pa': 'Punjabi',
  'pl': 'Polski',
  'ps': 'Pashto',
  'pt': 'Portugues',
  'ro': 'Romana',
  'ru': 'Русский',
  'sd': 'Sindhi',
  'si': 'Sinhala',
  'sk': 'Slovencina',
  'sl': 'Slovenscina',
  'sq': 'Shqip',
  'sr': 'Српски',
  'sv': 'Svenska',
  'sw': 'Kiswahili',
  'ta': 'Tamil',
  'te': 'Telugu',
  'th': 'ไทย',
  'tl': 'Tagalog',
  'tr': 'Türkce',
  'ug': 'Uyghur',
  'uk': 'Українська',
  'ur': 'Urdu',
  'uz': 'Ozbek',
  'vi': 'Tieng Viet',
  'zh': '中文',
  'zu': 'Zulu',
};
