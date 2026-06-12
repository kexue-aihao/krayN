import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:krayn/src/services/subscription_importer.dart';

void main() {
  const profileJson = {
    'id': 'node-1',
    'name': 'Test Node',
    'group': 'default',
    'transport': 'tcp',
    'endpoint': 'example.com:443',
    'client_id': 'client-id',
    'client_secret': 'client-secret',
    'server_public_key': 'server-public-key',
    'server_name': 'example.com',
    'handshake_padding': {'min': 0, 'max': 64},
  };

  test('parses native krayN config subscriptions', () {
    final profiles = SubscriptionImporter.parseProfiles(
      jsonEncode({
        'version': 1,
        'profiles': [profileJson],
      }),
    );

    expect(profiles, hasLength(1));
    expect(profiles.single.id, 'node-1');
    expect(profiles.single.name, 'Test Node');
    expect(profiles.single.endpoint, 'example.com:443');
  });

  test('parses base64-wrapped profile arrays', () {
    final body = base64.encode(utf8.encode(jsonEncode([profileJson])));

    final profiles = SubscriptionImporter.parseProfiles(body);

    expect(profiles, hasLength(1));
    expect(profiles.single.clientId, 'client-id');
  });

  test('prefers KLESS client secret fields from Kboard subscriptions', () {
    final profiles = SubscriptionImporter.parseProfiles(
      jsonEncode({
        'profiles': [
          {
            ...profileJson,
            'client_id': null,
            'user_id': 1001,
            'client_secret': 'panel-secret',
            'kless_client_secret': 'kless-secret',
          },
        ],
      }),
    );

    expect(profiles, hasLength(1));
    expect(profiles.single.clientId, '1001');
    expect(profiles.single.clientSecret, 'kless-secret');
  });

  test('rejects unsupported subscriptions', () {
    expect(
      () => SubscriptionImporter.parseProfiles('not a subscription'),
      throwsA(isA<SubscriptionImportException>()),
    );
  });
}
