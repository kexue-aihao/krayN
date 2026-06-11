import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../i18n/krayn_localizations.dart';
import '../models/diagnostic_result.dart';
import '../models/local_config.dart';
import '../models/profile.dart';
import '../services/core_api.dart';

class DiagnosticsPanel extends StatefulWidget {
  const DiagnosticsPanel({
    super.key,
    required this.api,
    required this.profile,
    required this.busy,
    required this.onMessage,
    required this.onError,
  });

  final CoreApi api;
  final Profile profile;
  final bool busy;
  final ValueChanged<String> onMessage;
  final ValueChanged<String> onError;

  @override
  State<DiagnosticsPanel> createState() => _DiagnosticsPanelState();
}

class _DiagnosticsPanelState extends State<DiagnosticsPanel> {
  static const _custom = '__custom__';
  static const _latencyOptions = [
    'https://www.google.com/generate_204',
    'https://www.gstatic.com/generate_204',
    'https://www.apple.com/library/test/success.html',
    'http://www.msftconnecttest.com/connecttest.txt',
  ];
  static const _speedOptions = [
    'https://cachefly.cachefly.net/50mb.test',
    'https://speed.cloudflare.com/__down?bytes=10000000',
    'https://speed.cloudflare.com/__down?bytes=50000000',
    'https://speed.cloudflare.com/__down?bytes=100000000',
  ];
  static const _dnsPresets = {
    '腾讯云 DNS': '119.29.29.29:53',
    '阿里云 DNS': '223.5.5.5:53',
    '114DNS': '114.114.114.114:53',
  };
  static const _dohPresets = {
    '腾讯云 DoH': 'https://doh.pub/dns-query',
    '阿里云 DoH': 'https://dns.alidns.com/dns-query',
  };

  late String _latencySelection;
  late String _speedSelection;
  late final TextEditingController _customLatency;
  late final TextEditingController _customSpeed;
  late final TextEditingController _resolverAddress;
  String _resolverType = 'system';
  String _resolverPreset = '';
  DiagnosticResult? _result;
  bool _testing = false;
  bool _loadingResolver = true;

  @override
  void initState() {
    super.initState();
    _latencySelection = _latencyOptions.first;
    _speedSelection = _speedOptions[1];
    _customLatency = TextEditingController();
    _customSpeed = TextEditingController();
    _resolverAddress = TextEditingController();
    unawaited(_loadResolver());
  }

  @override
  void dispose() {
    _customLatency.dispose();
    _customSpeed.dispose();
    _resolverAddress.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l = KrayNLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: ListView(
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(l.diagnostics, style: theme.textTheme.titleLarge),
                ),
                FilledButton.icon(
                  onPressed:
                      widget.busy || _testing || widget.profile.id.isEmpty
                          ? null
                          : _runDiagnostics,
                  icon: _testing
                      ? const SizedBox.square(
                          dimension: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.speed_outlined),
                  label: Text(_testing ? l.testing : l.runDiagnostics),
                ),
              ],
            ),
            const SizedBox(height: 16),
            _Selector(
              label: l.latencyTarget,
              icon: Icons.network_ping_outlined,
              value: _latencySelection,
              values: _latencyOptions,
              customLabel: l.customUrl,
              onChanged: (value) => setState(() => _latencySelection = value),
            ),
            if (_latencySelection == _custom) ...[
              const SizedBox(height: 10),
              TextField(
                controller: _customLatency,
                decoration: InputDecoration(
                  labelText: l.customUrl,
                  prefixIcon: const Icon(Icons.link),
                ),
                keyboardType: TextInputType.url,
              ),
            ],
            const SizedBox(height: 12),
            _Selector(
              label: l.speedTarget,
              icon: Icons.download_outlined,
              value: _speedSelection,
              values: _speedOptions,
              customLabel: l.customUrl,
              onChanged: (value) => setState(() => _speedSelection = value),
            ),
            if (_speedSelection == _custom) ...[
              const SizedBox(height: 10),
              TextField(
                controller: _customSpeed,
                decoration: InputDecoration(
                  labelText: l.customUrl,
                  prefixIcon: const Icon(Icons.link),
                ),
                keyboardType: TextInputType.url,
              ),
            ],
            const SizedBox(height: 16),
            _ResolverEditor(
              loading: _loadingResolver,
              type: _resolverType,
              preset: _resolverPreset,
              address: _resolverAddress,
              onTypeChanged: _setResolverType,
              onPresetChanged: _setResolverPreset,
              onSave: widget.busy ? null : _saveResolver,
            ),
            const SizedBox(height: 16),
            _ResultView(
              result: _result,
              onCopy: _copyText,
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _loadResolver() async {
    try {
      final local = await widget.api.getLocalConfig();
      if (!mounted) {
        return;
      }
      setState(() {
        _resolverType =
            local.resolverType.isEmpty ? 'system' : local.resolverType;
        _resolverAddress.text = local.resolverAddress;
        _resolverPreset = _presetFor(_resolverType, local.resolverAddress);
        _loadingResolver = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _loadingResolver = false);
      widget.onError('$error');
    }
  }

  Future<void> _runDiagnostics() async {
    setState(() => _testing = true);
    try {
      final result = await widget.api.runDiagnostics(
        widget.profile.id,
        latencyUrl: _selectedLatencyUrl(),
        speedUrl: _selectedSpeedUrl(),
      );
      if (mounted) {
        setState(() => _result = result);
      }
    } catch (error) {
      widget.onError('$error');
    } finally {
      if (mounted) {
        setState(() => _testing = false);
      }
    }
  }

  Future<void> _saveResolver() async {
    try {
      final current = await widget.api.getLocalConfig();
      final next = LocalConfig(
        apiAddress: current.apiAddress,
        socksAddress: current.socksAddress,
        allowLan: current.allowLan,
        mode: current.mode,
        systemProxyMode: current.systemProxyMode,
        resolverType: _resolverType,
        resolverAddress:
            _resolverType == 'system' ? '' : _resolverAddress.text.trim(),
      );
      await widget.api.updateLocalConfig(next);
      if (mounted) {
        widget.onMessage(KrayNLocalizations.of(context).saveResolver);
      }
    } catch (error) {
      widget.onError('$error');
    }
  }

  void _setResolverType(String type) {
    setState(() {
      _resolverType = type;
      _resolverPreset = '';
      if (type == 'system') {
        _resolverAddress.clear();
      } else {
        final presets = type == 'doh' ? _dohPresets : _dnsPresets;
        _resolverPreset = presets.keys.first;
        _resolverAddress.text = presets.values.first;
      }
    });
  }

  void _setResolverPreset(String preset) {
    setState(() {
      _resolverPreset = preset;
      if (preset == _custom) {
        return;
      }
      final presets = _resolverType == 'doh' ? _dohPresets : _dnsPresets;
      _resolverAddress.text = presets[preset] ?? '';
    });
  }

  String _selectedLatencyUrl() {
    if (_latencySelection == _custom) {
      return _customLatency.text.trim();
    }
    return _latencySelection;
  }

  String _selectedSpeedUrl() {
    if (_speedSelection == _custom) {
      return _customSpeed.text.trim();
    }
    return _speedSelection;
  }

  String _presetFor(String type, String address) {
    final presets = type == 'doh' ? _dohPresets : _dnsPresets;
    for (final entry in presets.entries) {
      if (entry.value == address) {
        return entry.key;
      }
    }
    return address.isEmpty ? '' : _custom;
  }

  Future<void> _copyText(String value) async {
    if (value.isEmpty) {
      return;
    }
    await Clipboard.setData(ClipboardData(text: value));
    if (mounted) {
      widget.onMessage(KrayNLocalizations.of(context).copied);
    }
  }
}

class _Selector extends StatelessWidget {
  const _Selector({
    required this.label,
    required this.icon,
    required this.value,
    required this.values,
    required this.customLabel,
    required this.onChanged,
  });

  final String label;
  final IconData icon;
  final String value;
  final List<String> values;
  final String customLabel;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return DropdownButtonFormField<String>(
      initialValue: value,
      decoration: InputDecoration(
        labelText: label,
        prefixIcon: Icon(icon),
      ),
      items: [
        ...values
            .map((item) => DropdownMenuItem(value: item, child: Text(item))),
        DropdownMenuItem(
            value: _DiagnosticsPanelState._custom, child: Text(customLabel)),
      ],
      onChanged: (value) {
        if (value != null) {
          onChanged(value);
        }
      },
    );
  }
}

class _ResolverEditor extends StatelessWidget {
  const _ResolverEditor({
    required this.loading,
    required this.type,
    required this.preset,
    required this.address,
    required this.onTypeChanged,
    required this.onPresetChanged,
    required this.onSave,
  });

  final bool loading;
  final String type;
  final String preset;
  final TextEditingController address;
  final ValueChanged<String> onTypeChanged;
  final ValueChanged<String> onPresetChanged;
  final VoidCallback? onSave;

  @override
  Widget build(BuildContext context) {
    final l = KrayNLocalizations.of(context);
    final presets = type == 'doh'
        ? _DiagnosticsPanelState._dohPresets
        : _DiagnosticsPanelState._dnsPresets;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l.resolverSettings,
            style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 12),
        SegmentedButton<String>(
          segments: [
            ButtonSegment(
              value: 'system',
              icon: const Icon(Icons.computer_outlined),
              label: Text(l.resolverSystem),
            ),
            ButtonSegment(
              value: 'dns',
              icon: const Icon(Icons.dns_outlined),
              label: Text(l.resolverDns),
            ),
            ButtonSegment(
              value: 'doh',
              icon: const Icon(Icons.https_outlined),
              label: Text(l.resolverDoh),
            ),
          ],
          selected: {type},
          onSelectionChanged:
              loading ? null : (values) => onTypeChanged(values.first),
        ),
        if (type != 'system') ...[
          const SizedBox(height: 12),
          DropdownButtonFormField<String>(
            initialValue: preset.isEmpty ? presets.keys.first : preset,
            decoration: InputDecoration(
              labelText: l.resolverPreset,
              prefixIcon: const Icon(Icons.list_outlined),
            ),
            items: [
              ...presets.keys.map(
                  (item) => DropdownMenuItem(value: item, child: Text(item))),
              DropdownMenuItem(
                value: _DiagnosticsPanelState._custom,
                child: Text(l.resolverCustom),
              ),
            ],
            onChanged: loading
                ? null
                : (value) {
                    if (value != null) {
                      onPresetChanged(value);
                    }
                  },
          ),
          const SizedBox(height: 12),
          TextField(
            controller: address,
            decoration: InputDecoration(
              labelText: l.resolverAddress,
              prefixIcon: const Icon(Icons.link_outlined),
            ),
          ),
        ],
        const SizedBox(height: 12),
        Align(
          alignment: Alignment.centerRight,
          child: OutlinedButton.icon(
            onPressed: loading ? null : onSave,
            icon: const Icon(Icons.save_outlined),
            label: Text(l.saveResolver),
          ),
        ),
      ],
    );
  }
}

class _ResultView extends StatelessWidget {
  const _ResultView({
    required this.result,
    required this.onCopy,
  });

  final DiagnosticResult? result;
  final ValueChanged<String> onCopy;

  @override
  Widget build(BuildContext context) {
    final l = KrayNLocalizations.of(context);
    final value = result;
    if (value == null) {
      return _EmptyResult(text: l.noDiagnosticResult);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            _Metric(
                label: l.rttAverage,
                value:
                    _ms(value.rttMs, available: value.rttSamplesMs.isNotEmpty),
                icon: Icons.router_outlined),
            _Metric(
                label: l.rttMax,
                value: _ms(value.rttMaxMs,
                    available: value.rttSamplesMs.isNotEmpty),
                icon: Icons.trending_up_outlined),
            _Metric(
                label: l.jitter,
                value: _ms(value.jitterMs,
                    available: value.rttSamplesMs.isNotEmpty),
                icon: Icons.waves_outlined),
            _Metric(
                label: l.packetLoss,
                value: _percent(value.packetLossPercent),
                icon: Icons.signal_wifi_bad_outlined),
            _Metric(
                label: l.httpsLatency,
                value: _ms(value.httpsMs),
                icon: Icons.lock_clock_outlined),
            _Metric(
              label: l.speedTest,
              value: value.speedMbps > 0
                  ? '${value.speedMbps.toStringAsFixed(2)} Mbps'
                  : l.unavailable,
              icon: Icons.speed_outlined,
            ),
          ],
        ),
        const SizedBox(height: 12),
        _InfoTile(
          icon: Icons.public,
          label: l.egressIp,
          value: value.egressIp.isEmpty ? l.unavailable : value.egressIp,
          onCopy: value.egressIp.isEmpty ? null : () => onCopy(value.egressIp),
        ),
        _InfoTile(
          icon: Icons.account_tree_outlined,
          label: l.asn,
          value: _asn(value, l),
          onCopy: value.asn == 0 ? null : () => onCopy(_asn(value, l)),
        ),
        _InfoTile(
          icon: Icons.verified_user_outlined,
          label: l.ipPurity,
          value: _purity(value, l),
          onCopy:
              value.purityUrl.isEmpty ? null : () => onCopy(value.purityUrl),
        ),
        _InfoTile(
          icon: Icons.dns_outlined,
          label: l.resolvedIps,
          value: value.resolvedIps.isEmpty
              ? l.unavailable
              : value.resolvedIps.join(', '),
          onCopy: value.resolvedIps.isEmpty
              ? null
              : () => onCopy(value.resolvedIps.join(', ')),
        ),
        _InfoTile(
          icon: Icons.network_check_outlined,
          label: l.rttSamples,
          value: value.rttSamplesMs.isEmpty
              ? l.unavailable
              : value.rttSamplesMs.map((sample) => '${sample}ms').join(', '),
          onCopy: value.rttSamplesMs.isEmpty
              ? null
              : () => onCopy(value.rttSamplesMs.join(', ')),
        ),
        _InfoTile(
          icon: Icons.hub_outlined,
          label: l.udpType,
          value: _udpType(value.udpType, l),
          onCopy: value.udpType.isEmpty ? null : () => onCopy(value.udpType),
        ),
        if (value.errors.isNotEmpty) ...[
          const SizedBox(height: 12),
          Text(l.diagnosticErrors,
              style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 6),
          ...value.errors.map(
            (error) => Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Text(error, style: Theme.of(context).textTheme.bodySmall),
            ),
          ),
        ],
      ],
    );
  }

  static String _ms(int value, {bool available = true}) {
    if (!available || value <= 0) {
      return '-';
    }
    return '$value ms';
  }

  static String _percent(double value) {
    if (value <= 0) {
      return '0%';
    }
    if (value >= 100) {
      return '100%';
    }
    return '${value.toStringAsFixed(value.truncateToDouble() == value ? 0 : 1)}%';
  }

  static String _udpType(String value, KrayNLocalizations l) {
    if (value.isEmpty) {
      return l.unavailable;
    }
    if (value == 'unsupported') {
      return l.udpUnsupported;
    }
    return value;
  }

  static String _asn(DiagnosticResult result, KrayNLocalizations l) {
    if (result.asn == 0 && result.asnOrganization.isEmpty) {
      return l.unavailable;
    }
    final parts = [
      if (result.asn > 0) 'AS${result.asn}',
      if (result.asnOrganization.isNotEmpty) result.asnOrganization,
      if (result.countryCode.isNotEmpty) result.countryCode,
    ];
    return parts.join('  ');
  }

  static String _purity(DiagnosticResult result, KrayNLocalizations l) {
    if (result.puritySummary.isNotEmpty) {
      return result.puritySummary;
    }
    if (result.purityUrl.isNotEmpty) {
      return result.purityUrl;
    }
    return l.unavailable;
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value, required this.icon});

  final String label;
  final String value;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: 160,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(icon),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, maxLines: 1, overflow: TextOverflow.ellipsis),
                const SizedBox(height: 3),
                Text(value, style: theme.textTheme.titleMedium),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _InfoTile extends StatelessWidget {
  const _InfoTile({
    required this.icon,
    required this.label,
    required this.value,
    this.onCopy,
  });

  final IconData icon;
  final String label;
  final String value;
  final VoidCallback? onCopy;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: Icon(icon),
      title: Text(label),
      subtitle: Text(value, maxLines: 2, overflow: TextOverflow.ellipsis),
      trailing: IconButton(
        tooltip: KrayNLocalizations.of(context).copied,
        onPressed: onCopy,
        icon: const Icon(Icons.copy_outlined),
      ),
    );
  }
}

class _EmptyResult extends StatelessWidget {
  const _EmptyResult({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 20),
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(text, textAlign: TextAlign.center),
    );
  }
}
