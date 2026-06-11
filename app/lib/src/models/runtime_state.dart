class RuntimeState {
  const RuntimeState({
    required this.status,
    required this.activeProfileId,
    required this.socksAddress,
    required this.apiAddress,
    required this.mode,
    required this.systemProxyMode,
    required this.uploadedBytes,
    required this.downloadedBytes,
    required this.activeConnections,
    required this.totalConnections,
    this.lastError = '',
  });

  final String status;
  final String lastError;
  final String activeProfileId;
  final String socksAddress;
  final String apiAddress;
  final String mode;
  final String systemProxyMode;
  final int uploadedBytes;
  final int downloadedBytes;
  final int activeConnections;
  final int totalConnections;

  bool get isRunning => status == 'running' || status == 'starting';

  factory RuntimeState.fromJson(Map<String, dynamic> json) {
    final stats = json['stats'] as Map<String, dynamic>? ?? {};
    return RuntimeState(
      status: json['status'] as String? ?? 'stopped',
      lastError: json['last_error'] as String? ?? '',
      activeProfileId: json['active_profile_id'] as String? ?? '',
      socksAddress: json['socks_address'] as String? ?? '',
      apiAddress: json['api_address'] as String? ?? '',
      mode: json['mode'] as String? ?? 'rule',
      systemProxyMode: json['system_proxy_mode'] as String? ?? 'unchanged',
      uploadedBytes: stats['uploaded_bytes'] as int? ?? 0,
      downloadedBytes: stats['downloaded_bytes'] as int? ?? 0,
      activeConnections: stats['active_connections'] as int? ?? 0,
      totalConnections: stats['total_connections'] as int? ?? 0,
    );
  }

  static const disconnected = RuntimeState(
    status: 'stopped',
    activeProfileId: '',
    socksAddress: '',
    apiAddress: '',
    mode: 'rule',
    systemProxyMode: 'unchanged',
    uploadedBytes: 0,
    downloadedBytes: 0,
    activeConnections: 0,
    totalConnections: 0,
  );
}
