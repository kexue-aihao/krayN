class LocalConfig {
  const LocalConfig({
    required this.apiAddress,
    required this.socksAddress,
    this.allowLan = false,
    this.mode = 'rule',
    this.systemProxyMode = 'unchanged',
    this.resolverType = 'system',
    this.resolverAddress = '',
  });

  final String apiAddress;
  final String socksAddress;
  final bool allowLan;
  final String mode;
  final String systemProxyMode;
  final String resolverType;
  final String resolverAddress;

  factory LocalConfig.fromJson(Map<String, dynamic> json) {
    return LocalConfig(
      apiAddress: json['api_address'] as String? ?? '127.0.0.1:9727',
      socksAddress: json['socks_address'] as String? ?? '127.0.0.1:7890',
      allowLan: json['allow_lan'] as bool? ?? false,
      mode: json['mode'] as String? ?? 'rule',
      systemProxyMode: json['system_proxy_mode'] as String? ?? 'unchanged',
      resolverType: json['resolver_type'] as String? ?? 'system',
      resolverAddress: json['resolver_address'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'api_address': apiAddress,
      'socks_address': socksAddress,
      'allow_lan': allowLan,
      'mode': mode,
      'system_proxy_mode': systemProxyMode,
      'resolver_type': resolverType,
      'resolver_address': resolverAddress,
    };
  }

  LocalConfig copyWith({
    String? apiAddress,
    String? socksAddress,
    bool? allowLan,
    String? mode,
    String? systemProxyMode,
    String? resolverType,
    String? resolverAddress,
  }) {
    return LocalConfig(
      apiAddress: apiAddress ?? this.apiAddress,
      socksAddress: socksAddress ?? this.socksAddress,
      allowLan: allowLan ?? this.allowLan,
      mode: mode ?? this.mode,
      systemProxyMode: systemProxyMode ?? this.systemProxyMode,
      resolverType: resolverType ?? this.resolverType,
      resolverAddress: resolverAddress ?? this.resolverAddress,
    );
  }
}
