import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/profile.dart';

class SubscriptionImporter {
  SubscriptionImporter({http.Client? client})
      : _client = client ?? http.Client(),
        _ownsClient = client == null;

  final http.Client _client;
  final bool _ownsClient;

  Future<List<Profile>> fetchProfiles(String input) async {
    final uri = _parseUrl(input);
    final response = await _fetch(uri);
    final body = utf8.decode(response.bodyBytes, allowMalformed: true);
    return parseProfiles(body);
  }

  void close() {
    if (_ownsClient) {
      _client.close();
    }
  }

  static List<Profile> parseProfiles(String body) {
    final text = body.trim();
    if (text.isEmpty) {
      throw const SubscriptionImportException(
        SubscriptionImportError.emptySubscription,
      );
    }

    final decoded = _decodeJsonLike(text);
    final profilesJson = _extractProfiles(decoded);
    final profiles = profilesJson.map(_profileFromJson).toList(growable: false);
    if (profiles.isEmpty) {
      throw const SubscriptionImportException(
        SubscriptionImportError.noProfiles,
      );
    }
    return profiles;
  }

  Uri _parseUrl(String input) {
    final text = input.trim();
    final uri = Uri.tryParse(text);
    if (uri == null || !uri.hasScheme || uri.host.isEmpty) {
      throw const SubscriptionImportException(
        SubscriptionImportError.invalidUrl,
      );
    }
    if (uri.scheme != 'http' && uri.scheme != 'https') {
      throw const SubscriptionImportException(
        SubscriptionImportError.invalidUrl,
      );
    }
    return uri;
  }

  Future<http.Response> _fetch(Uri uri) async {
    try {
      final response = await _client.get(
        uri,
        headers: const {
          'Accept': 'application/json, text/plain;q=0.9, */*;q=0.8',
          'User-Agent': 'krayN/0.1 subscription-import',
        },
      ).timeout(const Duration(seconds: 20));
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw SubscriptionImportException(
          SubscriptionImportError.requestFailed,
          statusCode: response.statusCode,
        );
      }
      return response;
    } on TimeoutException {
      throw const SubscriptionImportException(
        SubscriptionImportError.requestTimeout,
      );
    } on http.ClientException {
      throw const SubscriptionImportException(
        SubscriptionImportError.requestFailed,
      );
    }
  }

  static dynamic _decodeJsonLike(String text) {
    for (final candidate in [text, _tryDecodeBase64(text)]) {
      if (candidate == null || candidate.trim().isEmpty) {
        continue;
      }
      try {
        return jsonDecode(candidate);
      } on FormatException {
        continue;
      }
    }
    throw const SubscriptionImportException(
      SubscriptionImportError.unsupportedFormat,
    );
  }

  static String? _tryDecodeBase64(String text) {
    final compact = text.replaceAll(RegExp(r'\s+'), '');
    if (compact.length < 8 || compact.contains('{') || compact.contains('[')) {
      return null;
    }
    try {
      final padded = compact.padRight(
        compact.length + ((4 - compact.length % 4) % 4),
        '=',
      );
      return utf8.decode(base64.decode(padded), allowMalformed: true);
    } on FormatException {
      try {
        final padded = compact.padRight(
          compact.length + ((4 - compact.length % 4) % 4),
          '=',
        );
        return utf8.decode(base64Url.decode(padded), allowMalformed: true);
      } on FormatException {
        return null;
      }
    }
  }

  static List<Map<String, dynamic>> _extractProfiles(dynamic decoded) {
    final dynamic rawProfiles;
    if (decoded is List<dynamic>) {
      rawProfiles = decoded;
    } else if (decoded is Map<String, dynamic>) {
      rawProfiles =
          decoded['profiles'] ?? decoded['nodes'] ?? decoded['profile'];
    } else {
      throw const SubscriptionImportException(
        SubscriptionImportError.unsupportedFormat,
      );
    }

    if (rawProfiles is Map<String, dynamic>) {
      return [rawProfiles];
    }
    if (rawProfiles is! List<dynamic>) {
      throw const SubscriptionImportException(
        SubscriptionImportError.unsupportedFormat,
      );
    }
    return rawProfiles.whereType<Map<String, dynamic>>().toList(
          growable: false,
        );
  }

  static Profile _profileFromJson(Map<String, dynamic> json) {
    final profile = Profile.fromJson(json);
    if (profile.name.trim().isEmpty ||
        profile.endpoint.trim().isEmpty ||
        profile.clientId.trim().isEmpty ||
        profile.clientSecret.trim().isEmpty ||
        profile.serverPublicKey.trim().isEmpty) {
      throw const SubscriptionImportException(
        SubscriptionImportError.invalidProfile,
      );
    }
    return profile;
  }
}

class SubscriptionImportException implements Exception {
  const SubscriptionImportException(this.error, {this.statusCode});

  final SubscriptionImportError error;
  final int? statusCode;
}

enum SubscriptionImportError {
  invalidUrl,
  requestFailed,
  requestTimeout,
  emptySubscription,
  noProfiles,
  unsupportedFormat,
  invalidProfile,
}
