import 'package:flutter/material.dart';

import '../i18n/krayn_localizations.dart';
import '../models/profile.dart';

class ProfileEditor extends StatefulWidget {
  const ProfileEditor({
    super.key,
    required this.initialProfile,
    required this.busy,
    required this.onSave,
    required this.onDelete,
  });

  final Profile initialProfile;
  final bool busy;
  final ValueChanged<Profile> onSave;
  final ValueChanged<Profile> onDelete;

  @override
  State<ProfileEditor> createState() => _ProfileEditorState();
}

class _ProfileEditorState extends State<ProfileEditor> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _name;
  late final TextEditingController _group;
  late final TextEditingController _endpoint;
  late final TextEditingController _clientId;
  late final TextEditingController _clientSecret;
  late final TextEditingController _serverPublicKey;
  late final TextEditingController _serverName;
  late final TextEditingController _headers;
  late final TextEditingController _paddingMin;
  late final TextEditingController _paddingMax;
  late final TextEditingController _remark;
  late String _transport;
  late bool _skipTlsVerify;

  static const _transports = [
    'tcp',
    'tls',
    'websocket',
    'http-upgrade',
    'http-stream',
    'grpc',
    'xhttp',
  ];

  @override
  void initState() {
    super.initState();
    final profile = widget.initialProfile;
    _name = TextEditingController(text: profile.name);
    _group = TextEditingController(text: profile.group);
    _endpoint = TextEditingController(text: profile.endpoint);
    _clientId = TextEditingController(text: profile.clientId);
    _clientSecret = TextEditingController(text: profile.clientSecret);
    _serverPublicKey = TextEditingController(text: profile.serverPublicKey);
    _serverName = TextEditingController(text: profile.serverName);
    _headers = TextEditingController(text: _headersToText(profile.headers));
    _paddingMin = TextEditingController(text: '${profile.paddingMin}');
    _paddingMax = TextEditingController(text: '${profile.paddingMax}');
    _remark = TextEditingController(text: profile.remark);
    _transport = _transports.contains(profile.transport) ? profile.transport : 'tcp';
    _skipTlsVerify = profile.skipTlsVerify;
  }

  @override
  void dispose() {
    _name.dispose();
    _group.dispose();
    _endpoint.dispose();
    _clientId.dispose();
    _clientSecret.dispose();
    _serverPublicKey.dispose();
    _serverName.dispose();
    _headers.dispose();
    _paddingMin.dispose();
    _paddingMax.dispose();
    _remark.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l = KrayNLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      widget.initialProfile.id.isEmpty ? l.newNode : l.nodeSettings,
                      style: theme.textTheme.titleLarge,
                    ),
                  ),
                  IconButton(
                    tooltip: l.delete,
                    onPressed: widget.busy ? null : () => widget.onDelete(widget.initialProfile),
                    icon: const Icon(Icons.delete_outline),
                  ),
                  const SizedBox(width: 8),
                  FilledButton.icon(
                    onPressed: widget.busy ? null : _submit,
                    icon: const Icon(Icons.save_outlined),
                    label: Text(l.save),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              Expanded(
                child: ListView(
                  children: [
                    LayoutBuilder(
                      builder: (context, constraints) {
                        final twoColumns = constraints.maxWidth >= 680;
                        return Wrap(
                          spacing: 12,
                          runSpacing: 12,
                          children: [
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _name,
                                decoration: InputDecoration(
                                  labelText: l.name,
                                  prefixIcon: const Icon(Icons.label_outline),
                                ),
                                validator: (value) => _required(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _group,
                                decoration: InputDecoration(
                                  labelText: l.group,
                                  prefixIcon: const Icon(Icons.folder_outlined),
                                ),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: DropdownButtonFormField<String>(
                                initialValue: _transport,
                                decoration: InputDecoration(
                                  labelText: l.transport,
                                  prefixIcon: const Icon(Icons.route_outlined),
                                ),
                                items: _transports
                                    .map((item) => DropdownMenuItem(value: item, child: Text(item)))
                                    .toList(),
                                onChanged: (value) {
                                  if (value != null) {
                                    setState(() => _transport = value);
                                  }
                                },
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _endpoint,
                                decoration: InputDecoration(
                                  labelText: l.endpoint,
                                  prefixIcon: const Icon(Icons.public),
                                ),
                                validator: (value) => _required(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _clientId,
                                decoration: InputDecoration(
                                  labelText: l.clientId,
                                  prefixIcon: const Icon(Icons.badge_outlined),
                                ),
                                validator: (value) => _required(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _serverName,
                                decoration: InputDecoration(
                                  labelText: l.tlsServerName,
                                  prefixIcon: const Icon(Icons.dns_outlined),
                                ),
                              ),
                            ),
                            _FieldBox(
                              wide: false,
                              child: TextFormField(
                                controller: _clientSecret,
                                decoration: InputDecoration(
                                  labelText: l.clientSecret,
                                  prefixIcon: const Icon(Icons.key_outlined),
                                ),
                                validator: (value) => _required(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: false,
                              child: TextFormField(
                                controller: _serverPublicKey,
                                decoration: InputDecoration(
                                  labelText: l.serverPublicKey,
                                  prefixIcon: const Icon(Icons.verified_user_outlined),
                                ),
                                validator: (value) => _required(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _paddingMin,
                                keyboardType: TextInputType.number,
                                decoration: InputDecoration(
                                  labelText: l.paddingMin,
                                  prefixIcon: const Icon(Icons.compress_outlined),
                                ),
                                validator: (value) => _number(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: twoColumns,
                              child: TextFormField(
                                controller: _paddingMax,
                                keyboardType: TextInputType.number,
                                decoration: InputDecoration(
                                  labelText: l.paddingMax,
                                  prefixIcon: const Icon(Icons.unfold_more),
                                ),
                                validator: (value) => _number(value, l),
                              ),
                            ),
                            _FieldBox(
                              wide: false,
                              child: TextFormField(
                                controller: _headers,
                                maxLines: 4,
                                decoration: InputDecoration(
                                  labelText: l.headers,
                                  prefixIcon: const Icon(Icons.list_alt_outlined),
                                  alignLabelWithHint: true,
                                ),
                              ),
                            ),
                            _FieldBox(
                              wide: false,
                              child: TextFormField(
                                controller: _remark,
                                maxLines: 3,
                                decoration: InputDecoration(
                                  labelText: l.remark,
                                  prefixIcon: const Icon(Icons.notes_outlined),
                                  alignLabelWithHint: true,
                                ),
                              ),
                            ),
                          ],
                        );
                      },
                    ),
                    const SizedBox(height: 12),
                    SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(l.skipTlsVerification),
                      value: _skipTlsVerify,
                      onChanged: (value) => setState(() => _skipTlsVerify = value),
                      secondary: const Icon(Icons.shield_outlined),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    final paddingMin = int.tryParse(_paddingMin.text.trim()) ?? 0;
    final paddingMax = int.tryParse(_paddingMax.text.trim()) ?? 0;
    final profile = widget.initialProfile.copyWith(
      name: _name.text.trim(),
      group: _group.text.trim(),
      transport: _transport,
      endpoint: _endpoint.text.trim(),
      clientId: _clientId.text.trim(),
      clientSecret: _clientSecret.text.trim(),
      serverPublicKey: _serverPublicKey.text.trim(),
      serverName: _serverName.text.trim(),
      skipTlsVerify: _skipTlsVerify,
      headers: _parseHeaders(_headers.text),
      paddingMin: paddingMin,
      paddingMax: paddingMax,
      remark: _remark.text.trim(),
    );
    widget.onSave(profile);
  }

  String? _required(String? value, KrayNLocalizations l) {
    if (value == null || value.trim().isEmpty) {
      return l.requiredField;
    }
    return null;
  }

  String? _number(String? value, KrayNLocalizations l) {
    if (value == null || value.trim().isEmpty) {
      return null;
    }
    final parsed = int.tryParse(value.trim());
    if (parsed == null || parsed < 0) {
      return l.nonNegativeNumber;
    }
    return null;
  }

  static String _headersToText(Map<String, String> headers) {
    return headers.entries.map((entry) => '${entry.key}: ${entry.value}').join('\n');
  }

  static Map<String, String> _parseHeaders(String text) {
    final headers = <String, String>{};
    for (final raw in text.split('\n')) {
      final line = raw.trim();
      if (line.isEmpty) {
        continue;
      }
      final index = line.indexOf(':');
      if (index <= 0) {
        continue;
      }
      headers[line.substring(0, index).trim()] = line.substring(index + 1).trim();
    }
    return headers;
  }
}

class _FieldBox extends StatelessWidget {
  const _FieldBox({required this.child, required this.wide});

  final Widget child;
  final bool wide;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: wide ? 320 : double.infinity,
      child: child,
    );
  }
}
