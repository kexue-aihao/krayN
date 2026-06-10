import 'dart:async';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';

import '../models/profile.dart';
import '../models/runtime_state.dart';
import '../services/core_api.dart';
import '../services/core_process.dart';
import 'profile_editor.dart';
import 'widgets/status_panel.dart';

class KrayNApp extends StatelessWidget {
  const KrayNApp({super.key, required this.api, required this.process});

  final CoreApi api;
  final CoreProcess process;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'krayN',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xff006c67),
          brightness: Brightness.light,
        ),
        scaffoldBackgroundColor: const Color(0xfff7f8f4),
        cardTheme: const CardThemeData(
          elevation: 0,
          margin: EdgeInsets.zero,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.all(Radius.circular(8)),
          ),
        ),
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
          filled: true,
          fillColor: Colors.white,
        ),
      ),
      darkTheme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xff00a399),
          brightness: Brightness.dark,
        ),
        cardTheme: const CardThemeData(
          elevation: 0,
          margin: EdgeInsets.zero,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.all(Radius.circular(8)),
          ),
        ),
      ),
      home: DashboardPage(api: api, process: process),
    );
  }
}

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, required this.api, required this.process});

  final CoreApi api;
  final CoreProcess process;

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  RuntimeState _state = RuntimeState.disconnected;
  List<Profile> _profiles = const [];
  Profile _draft = Profile.empty;
  bool _loading = true;
  bool _busy = false;
  String _error = '';
  Timer? _timer;

  Profile? get _activeProfile {
    return _profiles.firstWhereOrNull((item) => item.id == _state.activeProfileId);
  }

  @override
  void initState() {
    super.initState();
    _boot();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) => _refresh(silent: true));
  }

  @override
  void dispose() {
    _timer?.cancel();
    widget.process.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    await widget.process.ensureStarted();
    await _refresh();
  }

  Future<void> _refresh({bool silent = false}) async {
    if (!silent) {
      setState(() {
        _loading = true;
        _error = '';
      });
    }
    try {
      final results = await Future.wait([
        widget.api.getState(),
        widget.api.getProfiles(),
      ]);
      final state = results[0] as RuntimeState;
      final profiles = results[1] as List<Profile>;
      if (!mounted) {
        return;
      }
      setState(() {
        _state = state;
        _profiles = profiles;
        if (_draft == Profile.empty && profiles.isNotEmpty) {
          _draft = profiles.first;
        }
        _loading = false;
        _error = '';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _loading = false;
        if (!silent) {
          _error = '$error';
        }
      });
    }
  }

  Future<void> _startStop() async {
    await _runBusy(() async {
      _state = _state.isRunning ? await widget.api.stop() : await widget.api.start();
      await _refresh(silent: true);
    });
  }

  Future<void> _activate(Profile profile) async {
    await _runBusy(() async {
      _state = await widget.api.activateProfile(profile.id);
      _draft = profile;
    });
  }

  Future<void> _save(Profile profile) async {
    await _runBusy(() async {
      final saved = await widget.api.saveProfile(profile);
      await _refresh(silent: true);
      _draft = saved;
    });
  }

  Future<void> _delete(Profile profile) async {
    if (profile.id.isEmpty) {
      setState(() => _draft = Profile.empty);
      return;
    }
    await _runBusy(() async {
      await widget.api.deleteProfile(profile.id);
      await _refresh(silent: true);
      _draft = Profile.empty;
    });
  }

  Future<void> _runBusy(Future<void> Function() action) async {
    setState(() {
      _busy = true;
      _error = '';
    });
    try {
      await action();
    } catch (error) {
      setState(() => _error = '$error');
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final isWide = width >= 920;
    return Scaffold(
      appBar: AppBar(
        title: const Text('krayN'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            onPressed: _loading ? null : () => _refresh(),
            icon: const Icon(Icons.refresh),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              StatusPanel(
                state: _state,
                activeProfile: _activeProfile,
                busy: _busy,
                onStartStop: _startStop,
              ),
              if (_error.isNotEmpty) ...[
                const SizedBox(height: 12),
                _ErrorBanner(message: _error),
              ],
              const SizedBox(height: 16),
              Expanded(
                child: isWide
                    ? Row(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          SizedBox(
                            width: 350,
                            child: _ProfileList(
                              profiles: _profiles,
                              activeProfileId: _state.activeProfileId,
                              selectedId: _draft.id,
                              onSelect: (profile) => setState(() => _draft = profile),
                              onActivate: _activate,
                              onAdd: () => setState(() => _draft = Profile.empty),
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: ProfileEditor(
                              key: ValueKey(_draft.id),
                              initialProfile: _draft,
                              busy: _busy,
                              onSave: _save,
                              onDelete: _delete,
                            ),
                          ),
                        ],
                      )
                    : ListView(
                        children: [
                          SizedBox(
                            height: 430,
                            child: _ProfileList(
                              profiles: _profiles,
                              activeProfileId: _state.activeProfileId,
                              selectedId: _draft.id,
                              onSelect: (profile) => setState(() => _draft = profile),
                              onActivate: _activate,
                              onAdd: () => setState(() => _draft = Profile.empty),
                            ),
                          ),
                          const SizedBox(height: 16),
                          SizedBox(
                            height: 760,
                            child: ProfileEditor(
                              key: ValueKey(_draft.id),
                              initialProfile: _draft,
                              busy: _busy,
                              onSave: _save,
                              onDelete: _delete,
                            ),
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
}

class _ProfileList extends StatelessWidget {
  const _ProfileList({
    required this.profiles,
    required this.activeProfileId,
    required this.selectedId,
    required this.onSelect,
    required this.onActivate,
    required this.onAdd,
  });

  final List<Profile> profiles;
  final String activeProfileId;
  final String selectedId;
  final ValueChanged<Profile> onSelect;
  final ValueChanged<Profile> onActivate;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    'Nodes',
                    style: theme.textTheme.titleLarge,
                  ),
                ),
                IconButton.filledTonal(
                  tooltip: 'Add node',
                  onPressed: onAdd,
                  icon: const Icon(Icons.add),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Expanded(
              child: profiles.isEmpty
                  ? const Center(
                      child: Text('No nodes yet'),
                    )
                  : ListView.separated(
                      itemBuilder: (context, index) {
                        final profile = profiles[index];
                        final selected = profile.id == selectedId;
                        final active = profile.id == activeProfileId;
                        return Material(
                          color: selected
                              ? theme.colorScheme.secondaryContainer
                              : Colors.transparent,
                          borderRadius: BorderRadius.circular(8),
                          child: ListTile(
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8),
                            ),
                            leading: CircleAvatar(
                              backgroundColor: active
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.surfaceContainerHighest,
                              foregroundColor: active
                                  ? theme.colorScheme.onPrimary
                                  : theme.colorScheme.onSurfaceVariant,
                              child: Icon(_transportIcon(profile.transport)),
                            ),
                            title: Text(
                              profile.name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                            subtitle: Text(
                              '${profile.transport}  ${profile.endpoint}',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                            trailing: IconButton(
                              tooltip: 'Activate',
                              onPressed: active ? null : () => onActivate(profile),
                              icon: Icon(active ? Icons.check_circle : Icons.radio_button_unchecked),
                            ),
                            onTap: () => onSelect(profile),
                          ),
                        );
                      },
                      separatorBuilder: (_, __) => const SizedBox(height: 8),
                      itemCount: profiles.length,
                    ),
            ),
          ],
        ),
      ),
    );
  }

  IconData _transportIcon(String transport) {
    switch (transport) {
      case 'websocket':
        return Icons.hub;
      case 'tls':
        return Icons.lock;
      case 'grpc':
        return Icons.account_tree;
      case 'xhttp':
      case 'http-stream':
      case 'http-upgrade':
        return Icons.http;
      default:
        return Icons.cable;
    }
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: theme.colorScheme.onErrorContainer),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: TextStyle(color: theme.colorScheme.onErrorContainer),
            ),
          ),
        ],
      ),
    );
  }
}
