class DiagnosticResult {
  const DiagnosticResult({
    required this.profileId,
    required this.profileName,
    required this.latencyUrl,
    required this.speedUrl,
    this.rttMs = 0,
    this.httpsMs = 0,
    this.speedMbps = 0,
    this.downloadedBytes = 0,
    this.egressIp = '',
    this.asn = 0,
    this.asnOrganization = '',
    this.isp = '',
    this.country = '',
    this.countryCode = '',
    this.city = '',
    this.puritySummary = '',
    this.purityUrl = '',
    this.resolvedIps = const [],
    this.errors = const [],
  });

  final String profileId;
  final String profileName;
  final int rttMs;
  final int httpsMs;
  final double speedMbps;
  final int downloadedBytes;
  final String egressIp;
  final int asn;
  final String asnOrganization;
  final String isp;
  final String country;
  final String countryCode;
  final String city;
  final String puritySummary;
  final String purityUrl;
  final List<String> resolvedIps;
  final String latencyUrl;
  final String speedUrl;
  final List<String> errors;

  factory DiagnosticResult.fromJson(Map<String, dynamic> json) {
    return DiagnosticResult(
      profileId: json['profile_id'] as String? ?? '',
      profileName: json['profile_name'] as String? ?? '',
      rttMs: json['rtt_ms'] as int? ?? 0,
      httpsMs: json['https_ms'] as int? ?? 0,
      speedMbps: (json['speed_mbps'] as num?)?.toDouble() ?? 0,
      downloadedBytes: json['downloaded_bytes'] as int? ?? 0,
      egressIp: json['egress_ip'] as String? ?? '',
      asn: json['asn'] as int? ?? 0,
      asnOrganization: json['asn_organization'] as String? ?? '',
      isp: json['isp'] as String? ?? '',
      country: json['country'] as String? ?? '',
      countryCode: json['country_code'] as String? ?? '',
      city: json['city'] as String? ?? '',
      puritySummary: json['purity_summary'] as String? ?? '',
      purityUrl: json['purity_url'] as String? ?? '',
      resolvedIps: (json['resolved_ips'] as List<dynamic>? ?? [])
          .map((e) => '$e')
          .toList(),
      latencyUrl: json['latency_url'] as String? ?? '',
      speedUrl: json['speed_url'] as String? ?? '',
      errors:
          (json['errors'] as List<dynamic>? ?? []).map((e) => '$e').toList(),
    );
  }
}
