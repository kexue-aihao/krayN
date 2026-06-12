class Profile {
  const Profile({
    required this.id,
    required this.name,
    required this.transport,
    required this.endpoint,
    required this.clientId,
    required this.clientSecret,
    required this.serverPublicKey,
    this.group = '',
    this.serverName = '',
    this.skipTlsVerify = false,
    this.headers = const {},
    this.paddingMin = 0,
    this.paddingMax = 0,
    this.tags = const [],
    this.remark = '',
  });

  final String id;
  final String name;
  final String group;
  final String transport;
  final String endpoint;
  final String clientId;
  final String clientSecret;
  final String serverPublicKey;
  final String serverName;
  final bool skipTlsVerify;
  final Map<String, String> headers;
  final int paddingMin;
  final int paddingMax;
  final List<String> tags;
  final String remark;

  factory Profile.fromJson(Map<String, dynamic> json) {
    final padding = json['handshake_padding'] as Map<String, dynamic>? ?? {};
    return Profile(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      group: json['group'] as String? ?? '',
      transport: json['transport'] as String? ?? 'tcp',
      endpoint: json['endpoint'] as String? ?? '',
      clientId: _firstJsonString(
        json,
        ['client_id', 'clientId', 'uuid', 'user_uuid', 'user_id'],
      ),
      clientSecret: _firstJsonString(
        json,
        ['kless_client_secret', 'klessClientSecret', 'client_secret', 'clientSecret'],
      ),
      serverPublicKey: json['server_public_key'] as String? ?? '',
      serverName: json['server_name'] as String? ?? '',
      skipTlsVerify: json['skip_tls_verify'] as bool? ?? false,
      headers: (json['headers'] as Map<String, dynamic>? ?? {})
          .map((key, value) => MapEntry(key, '$value')),
      paddingMin: padding['min'] as int? ?? 0,
      paddingMax: padding['max'] as int? ?? 0,
      tags: (json['tags'] as List<dynamic>? ?? []).map((e) => '$e').toList(),
      remark: json['remark'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      if (id.isNotEmpty) 'id': id,
      'name': name,
      if (group.isNotEmpty) 'group': group,
      'transport': transport,
      'endpoint': endpoint,
      'client_id': clientId,
      'client_secret': clientSecret,
      'server_public_key': serverPublicKey,
      if (serverName.isNotEmpty) 'server_name': serverName,
      'skip_tls_verify': skipTlsVerify,
      if (headers.isNotEmpty) 'headers': headers,
      'handshake_padding': {'min': paddingMin, 'max': paddingMax},
      if (tags.isNotEmpty) 'tags': tags,
      if (remark.isNotEmpty) 'remark': remark,
    };
  }

  Profile copyWith({
    String? id,
    String? name,
    String? group,
    String? transport,
    String? endpoint,
    String? clientId,
    String? clientSecret,
    String? serverPublicKey,
    String? serverName,
    bool? skipTlsVerify,
    Map<String, String>? headers,
    int? paddingMin,
    int? paddingMax,
    List<String>? tags,
    String? remark,
  }) {
    return Profile(
      id: id ?? this.id,
      name: name ?? this.name,
      group: group ?? this.group,
      transport: transport ?? this.transport,
      endpoint: endpoint ?? this.endpoint,
      clientId: clientId ?? this.clientId,
      clientSecret: clientSecret ?? this.clientSecret,
      serverPublicKey: serverPublicKey ?? this.serverPublicKey,
      serverName: serverName ?? this.serverName,
      skipTlsVerify: skipTlsVerify ?? this.skipTlsVerify,
      headers: headers ?? this.headers,
      paddingMin: paddingMin ?? this.paddingMin,
      paddingMax: paddingMax ?? this.paddingMax,
      tags: tags ?? this.tags,
      remark: remark ?? this.remark,
    );
  }

  static const empty = Profile(
    id: '',
    name: '',
    transport: 'tcp',
    endpoint: '',
    clientId: '',
    clientSecret: '',
    serverPublicKey: '',
  );

  static String _firstJsonString(Map<String, dynamic> json, List<String> keys) {
    for (final key in keys) {
      final value = json[key];
      if (value == null) {
        continue;
      }
      final text = '$value'.trim();
      if (text.isNotEmpty) {
        return text;
      }
    }
    return '';
  }
}

