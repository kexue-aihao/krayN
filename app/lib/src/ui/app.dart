import 'dart:async';
import 'dart:math' as math;

import 'package:collection/collection.dart';
import 'package:flutter/services.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart';

import '../i18n/krayn_localizations.dart';
import '../i18n/language_catalog.dart';
import '../models/local_config.dart';
import '../models/profile.dart';
import '../models/runtime_state.dart';
import '../services/core_api.dart';
import '../services/core_process.dart';
import '../services/desktop_tray.dart';
import '../services/subscription_importer.dart';
import '../services/system_proxy.dart';
import 'brand/krayn_logo_mark.dart';
import 'profile_editor.dart';
import 'widgets/status_panel.dart';

class KrayNApp extends StatefulWidget {
  const KrayNApp({super.key, required this.api, required this.process});

  final CoreApi api;
  final CoreProcess process;

  @override
  State<KrayNApp> createState() => _KrayNAppState();
}

class _KrayNAppState extends State<KrayNApp> {
  Locale _locale = KrayNLocalizations.defaultLocale;

  void _setLocale(Locale locale) {
    setState(() => _locale = locale);
  }

  @override
  Widget build(BuildContext context) {
    Intl.defaultLocale = _locale.toLanguageTag();
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'krayN',
      locale: _locale,
      supportedLocales: KrayNLocalizations.supportedLocales,
      localizationsDelegates: const [
        KrayNLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ],
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
      home: DashboardPage(
        api: widget.api,
        process: widget.process,
        locale: _locale,
        onLocaleChanged: _setLocale,
      ),
    );
  }
}

class DashboardPage extends StatefulWidget {
  const DashboardPage({
    super.key,
    required this.api,
    required this.process,
    required this.locale,
    required this.onLocaleChanged,
  });

  final CoreApi api;
  final CoreProcess process;
  final Locale locale;
  final ValueChanged<Locale> onLocaleChanged;

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  RuntimeState _state = RuntimeState.disconnected;
  List<Profile> _profiles = const [];
  Profile _draft = Profile.empty;
  final SubscriptionImporter _subscriptionImporter = SubscriptionImporter();
  final SystemProxy _systemProxy = SystemProxy();
  late final DesktopTrayController _trayController;
  bool _loading = true;
  bool _busy = false;
  String _error = '';
  Timer? _timer;

  Profile? get _activeProfile {
    return _profiles.firstWhereOrNull(
      (item) => item.id == _state.activeProfileId,
    );
  }

  @override
  void initState() {
    super.initState();
    _trayController = DesktopTrayController(
      onStart: _start,
      onStop: _stop,
      onActivateProfile: _activate,
      onRouteMode: _setRouteMode,
      onSystemProxyMode: _setSystemProxyMode,
      onImportSubscription: _importSubscriptionFromClipboard,
      onRefresh: () => _refresh(),
      onExit: _shutdownFromTray,
    );
    _boot();
    _timer = Timer.periodic(
      const Duration(seconds: 2),
      (_) => _refresh(silent: true),
    );
  }

  @override
  void dispose() {
    _timer?.cancel();
    unawaited(_trayController.dispose());
    _subscriptionImporter.close();
    widget.process.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    await widget.process.ensureStarted();
    try {
      await _trayController.init();
    } catch (_) {
      // Some Linux desktop sessions do not expose a tray host.
    }
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
      await _trayController.refresh(state, profiles);
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
    if (_state.isRunning) {
      await _stop();
    } else {
      await _start();
    }
  }

  Future<void> _start() async {
    await _runBusy(() async {
      _state = await widget.api.start();
      await _applySystemProxy(_state.systemProxyMode);
      await _refresh(silent: true);
    });
  }

  Future<void> _stop() async {
    await _runBusy(() async {
      await _clearManagedSystemProxy();
      _state = await widget.api.stop();
      await _refresh(silent: true);
    });
  }

  Future<void> _activate(Profile profile) async {
    await _runBusy(() async {
      _state = await widget.api.activateProfile(profile.id);
      _draft = profile;
      if (_state.isRunning) {
        await _applySystemProxy(_state.systemProxyMode);
      }
      await _trayController.refresh(_state, _profiles);
    });
  }

  Future<void> _save(Profile profile) async {
    await _runBusy(() async {
      final saved = await widget.api.saveProfile(profile);
      await _refresh(silent: true);
      _draft = saved;
    });
  }

  Future<void> _importSubscription(String url) async {
    final l = KrayNLocalizations.of(context);
    await _runBusy(() async {
      try {
        final profiles = await _subscriptionImporter.fetchProfiles(url);
        Profile? lastSaved;
        for (final profile in profiles) {
          lastSaved = await widget.api.saveProfile(profile);
        }
        await _refresh(silent: true);
        if (lastSaved != null) {
          _draft = lastSaved;
        }
        if (mounted) {
          _showMessage(l.subscriptionImported(profiles.length));
        }
      } on SubscriptionImportException catch (error) {
        throw _localizedSubscriptionError(l, error);
      }
    });
  }

  Future<void> _importSubscriptionFromClipboard() async {
    final l = KrayNLocalizations.of(context);
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final url = data?.text?.trim() ?? '';
    if (url.isEmpty) {
      throw Exception(l.clipboardEmpty);
    }
    await _importSubscription(url);
  }

  Future<void> _setRouteMode(String mode) async {
    await _updateLocal((local) => local.copyWith(mode: mode));
  }

  Future<void> _setSystemProxyMode(String mode) async {
    await _updateLocal((local) => local.copyWith(systemProxyMode: mode));
    await _applySystemProxy(mode);
  }

  Future<void> _updateLocal(
      LocalConfig Function(LocalConfig local) update) async {
    await _runBusy(() async {
      final local = await widget.api.getLocalConfig();
      await widget.api.updateLocalConfig(update(local));
      await _refresh(silent: true);
    });
  }

  Future<void> _applySystemProxy(String mode) async {
    final l = KrayNLocalizations.of(context);
    final pacUrl = 'http://${_state.apiAddress}/proxy.pac';
    try {
      await _systemProxy.apply(
        mode: mode,
        socksAddress: _state.socksAddress,
        pacUrl: pacUrl,
      );
    } on SystemProxyException catch (error) {
      throw Exception('${l.systemProxyApplyFailed}: ${error.message}');
    }
  }

  Future<void> _clearManagedSystemProxy() async {
    if (_state.systemProxyMode == 'auto' ||
        _state.systemProxyMode == 'pac' ||
        _state.systemProxyMode == 'clear') {
      await _systemProxy.clear();
    }
  }

  Future<void> _shutdownFromTray() async {
    await _clearManagedSystemProxy();
    if (_state.isRunning) {
      await widget.api.stop();
    }
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
    final l = KrayNLocalizations.of(context);
    final width = MediaQuery.sizeOf(context).width;
    final isWide = width >= 920;
    return Scaffold(
      appBar: AppBar(
        leadingWidth: 60,
        leading: const Padding(
          padding: EdgeInsets.only(left: 16),
          child: Center(child: KrayNLogoMark(size: 34)),
        ),
        title: Text(l.appTitle),
        actions: [
          IconButton(
            tooltip: l.language,
            onPressed: () => _showLanguageDialog(context),
            icon: const Icon(Icons.language),
          ),
          IconButton(
            tooltip: l.refresh,
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
                              onSelect: (profile) =>
                                  setState(() => _draft = profile),
                              onActivate: _activate,
                              onImport: () => _showSubscriptionDialog(context),
                              onAdd: () =>
                                  setState(() => _draft = Profile.empty),
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
                              onSelect: (profile) =>
                                  setState(() => _draft = profile),
                              onActivate: _activate,
                              onImport: () => _showSubscriptionDialog(context),
                              onAdd: () =>
                                  setState(() => _draft = Profile.empty),
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

  Future<void> _showSubscriptionDialog(BuildContext context) async {
    final controller = TextEditingController();
    final formKey = GlobalKey<FormState>();
    final l = KrayNLocalizations.of(context);
    final url = await showDialog<String>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: Text(l.importSubscription),
          content: SizedBox(
            width: 460,
            child: Form(
              key: formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TextFormField(
                    controller: controller,
                    autofocus: true,
                    minLines: 1,
                    maxLines: 3,
                    decoration: InputDecoration(
                      labelText: l.subscriptionUrl,
                      hintText: l.subscriptionUrlHint,
                      prefixIcon: const Icon(Icons.link),
                    ),
                    keyboardType: TextInputType.url,
                    validator: (value) {
                      final uri = Uri.tryParse(value?.trim() ?? '');
                      if (uri == null ||
                          !uri.hasScheme ||
                          uri.host.isEmpty ||
                          (uri.scheme != 'http' && uri.scheme != 'https')) {
                        return l.invalidSubscriptionUrl;
                      }
                      return null;
                    },
                  ),
                  const SizedBox(height: 12),
                  Text(
                    l.subscriptionImportHelp,
                    style: Theme.of(dialogContext).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: Text(l.close),
            ),
            FilledButton.icon(
              onPressed: () {
                if (formKey.currentState!.validate()) {
                  Navigator.of(dialogContext).pop(controller.text.trim());
                }
              },
              icon: const Icon(Icons.download_for_offline_outlined),
              label: Text(l.importAction),
            ),
          ],
        );
      },
    );
    controller.dispose();
    if (url == null || url.isEmpty) {
      return;
    }
    await _importSubscription(url);
  }

  Future<void> _showLanguageDialog(BuildContext context) {
    final selectedTag = widget.locale.toLanguageTag();
    return showDialog<void>(
      context: context,
      builder: (dialogContext) {
        final l = KrayNLocalizations.of(dialogContext);
        final dialogHeight = math.min(
          520.0,
          MediaQuery.sizeOf(dialogContext).height * 0.68,
        );
        return AlertDialog(
          title: Text(l.chooseLanguage),
          contentPadding: const EdgeInsets.fromLTRB(0, 12, 0, 0),
          content: SizedBox(
            width: 420,
            height: dialogHeight,
            child: ListView.separated(
              itemCount: LanguageCatalog.options.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final option = LanguageCatalog.options[index];
                final selected = option.tag == selectedTag;
                return ListTile(
                  selected: selected,
                  leading: Icon(selected ? Icons.check_circle : Icons.language),
                  title: Text(option.label),
                  subtitle: Text(option.tag),
                  onTap: () {
                    widget.onLocaleChanged(option.locale);
                    Navigator.of(dialogContext).pop();
                  },
                );
              },
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: Text(l.close),
            ),
          ],
        );
      },
    );
  }

  String _localizedSubscriptionError(
    KrayNLocalizations l,
    SubscriptionImportException error,
  ) {
    switch (error.error) {
      case SubscriptionImportError.invalidUrl:
        return l.invalidSubscriptionUrl;
      case SubscriptionImportError.requestFailed:
        if (error.statusCode != null) {
          return '${l.subscriptionRequestFailed} (HTTP ${error.statusCode})';
        }
        return l.subscriptionRequestFailed;
      case SubscriptionImportError.requestTimeout:
        return l.subscriptionRequestTimeout;
      case SubscriptionImportError.emptySubscription:
        return l.emptySubscription;
      case SubscriptionImportError.noProfiles:
        return l.noSubscriptionProfiles;
      case SubscriptionImportError.unsupportedFormat:
        return l.unsupportedSubscriptionFormat;
      case SubscriptionImportError.invalidProfile:
        return l.invalidSubscriptionProfile;
    }
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(message)));
  }
}

class _ProfileList extends StatelessWidget {
  const _ProfileList({
    required this.profiles,
    required this.activeProfileId,
    required this.selectedId,
    required this.onSelect,
    required this.onActivate,
    required this.onImport,
    required this.onAdd,
  });

  final List<Profile> profiles;
  final String activeProfileId;
  final String selectedId;
  final ValueChanged<Profile> onSelect;
  final ValueChanged<Profile> onActivate;
  final VoidCallback onImport;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l = KrayNLocalizations.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(l.nodes, style: theme.textTheme.titleLarge),
                ),
                IconButton.filledTonal(
                  tooltip: l.addNode,
                  onPressed: onAdd,
                  icon: const Icon(Icons.add),
                ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: onImport,
                icon: const Icon(Icons.link),
                label: Text(l.importSubscription),
              ),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: profiles.isEmpty
                  ? Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(l.noNodes),
                          const SizedBox(height: 12),
                          FilledButton.icon(
                            onPressed: onImport,
                            icon: const Icon(Icons.link),
                            label: Text(l.importSubscription),
                          ),
                        ],
                      ),
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
                              tooltip: active ? l.active : l.activate,
                              onPressed:
                                  active ? null : () => onActivate(profile),
                              icon: Icon(
                                active
                                    ? Icons.check_circle
                                    : Icons.radio_button_unchecked,
                              ),
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
