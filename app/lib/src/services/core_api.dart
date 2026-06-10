import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/profile.dart';
import '../models/runtime_state.dart';

class CoreApi {
  CoreApi({this.baseUrl = 'http://127.0.0.1:9727'});

  final String baseUrl;
  final http.Client _client = http.Client();

  Future<RuntimeState> getState() async {
    final json = await _getJson('/state');
    return RuntimeState.fromJson(json as Map<String, dynamic>);
  }

  Future<List<Profile>> getProfiles() async {
    final json = await _getJson('/profiles');
    return (json as List<dynamic>)
        .map((item) => Profile.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<Profile> saveProfile(Profile profile) async {
    final json = profile.id.isEmpty
        ? await _sendJson('POST', '/profiles', profile.toJson())
        : await _sendJson('PUT', '/profiles/${profile.id}', profile.toJson());
    return Profile.fromJson(json as Map<String, dynamic>);
  }

  Future<void> deleteProfile(String id) async {
    await _sendJson('DELETE', '/profiles/$id', null);
  }

  Future<RuntimeState> activateProfile(String id) async {
    final json = await _sendJson('POST', '/profiles/$id/activate', null);
    return RuntimeState.fromJson(json as Map<String, dynamic>);
  }

  Future<RuntimeState> start() async {
    final json = await _sendJson('POST', '/start', null);
    return RuntimeState.fromJson(json as Map<String, dynamic>);
  }

  Future<RuntimeState> stop() async {
    final json = await _sendJson('POST', '/stop', null);
    return RuntimeState.fromJson(json as Map<String, dynamic>);
  }

  Future<dynamic> _getJson(String path) async {
    final uri = Uri.parse('$baseUrl$path');
    final response = await _client.get(uri).timeout(const Duration(seconds: 5));
    return _decode(response);
  }

  Future<dynamic> _sendJson(
    String method,
    String path,
    Object? body,
  ) async {
    final uri = Uri.parse('$baseUrl$path');
    final request = http.Request(method, uri);
    request.headers['Content-Type'] = 'application/json; charset=utf-8';
    if (body != null) {
      request.body = jsonEncode(body);
    }
    final streamed = await _client.send(request).timeout(const Duration(seconds: 8));
    final response = await http.Response.fromStream(streamed);
    return _decode(response);
  }

  dynamic _decode(http.Response response) {
    final text = utf8.decode(response.bodyBytes);
    final json = text.isEmpty ? null : jsonDecode(text);
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return json;
    }
    if (json is Map<String, dynamic> && json['error'] != null) {
      throw CoreApiException(json['error'] as String);
    }
    throw CoreApiException('HTTP ${response.statusCode}: ${response.reasonPhrase}');
  }
}

class CoreApiException implements Exception {
  const CoreApiException(this.message);
  final String message;

  @override
  String toString() => message;
}

