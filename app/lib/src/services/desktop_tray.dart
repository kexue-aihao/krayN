import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:window_manager/window_manager.dart';

import '../models/profile.dart';
import '../models/runtime_state.dart';

typedef TrayProfileAction = Future<void> Function(Profile profile);
typedef TrayModeAction = Future<void> Function(String mode);
typedef TraySystemProxyAction = Future<void> Function(String mode);
typedef TrayVoidAction = Future<void> Function();

class DesktopTrayController with TrayListener, WindowListener {
  DesktopTrayController({
    required this.onStart,
    required this.onStop,
    required this.onActivateProfile,
    required this.onRouteMode,
    required this.onSystemProxyMode,
    required this.onImportSubscription,
    required this.onRefresh,
    required this.onExit,
  });

  final TrayVoidAction onStart;
  final TrayVoidAction onStop;
  final TrayProfileAction onActivateProfile;
  final TrayModeAction onRouteMode;
  final TraySystemProxyAction onSystemProxyMode;
  final TrayVoidAction onImportSubscription;
  final TrayVoidAction onRefresh;
  final TrayVoidAction onExit;

  bool _initialized = false;
  List<Profile> _profiles = const [];
  RuntimeState _state = RuntimeState.disconnected;

  static bool get isSupported {
    return !kIsWeb &&
        (Platform.isWindows || Platform.isMacOS || Platform.isLinux);
  }

  Future<void> init() async {
    if (!isSupported || _initialized) {
      return;
    }
    await windowManager.ensureInitialized();
    await windowManager.setPreventClose(true);
    await trayManager.setIcon(
      Platform.isWindows
          ? 'assets/brand/krayn-tray.ico'
          : 'assets/brand/krayn-icon.png',
    );
    await trayManager.setToolTip('krayN');
    trayManager.addListener(this);
    windowManager.addListener(this);
    _initialized = true;
    await refresh(_state, _profiles);
  }

  Future<void> refresh(RuntimeState state, List<Profile> profiles) async {
    _state = state;
    _profiles = List<Profile>.unmodifiable(profiles);
    if (!_initialized) {
      return;
    }
    await trayManager.setContextMenu(_buildMenu());
    await trayManager.setToolTip(
      state.isRunning ? 'krayN - ${_activeProfileName()}' : 'krayN - stopped',
    );
  }

  Future<void> dispose() async {
    if (!_initialized) {
      return;
    }
    trayManager.removeListener(this);
    windowManager.removeListener(this);
    await trayManager.destroy();
    _initialized = false;
  }

  @override
  void onTrayIconMouseDown() {
    unawaited(_showWindow());
  }

  @override
  void onTrayIconRightMouseDown() {
    unawaited(trayManager.popUpContextMenu());
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) {
    final key = menuItem.key;
    if (key == null) {
      return;
    }
    unawaited(_handleMenuItem(key));
  }

  @override
  void onWindowClose() {
    unawaited(windowManager.hide());
  }

  Future<void> _handleMenuItem(String key) async {
    switch (key) {
      case 'show_window':
        await _showWindow();
        return;
      case 'start':
        await onStart();
        return;
      case 'stop':
        await onStop();
        return;
      case 'import_subscription':
        await onImportSubscription();
        return;
      case 'refresh':
        await onRefresh();
        return;
      case 'exit':
        await onExit();
        await windowManager.setPreventClose(false);
        await trayManager.destroy();
        await windowManager.destroy();
        return;
    }
    if (key.startsWith('profile:')) {
      final id = key.substring('profile:'.length);
      final profile = _profiles.where((item) => item.id == id).firstOrNull;
      if (profile != null) {
        await onActivateProfile(profile);
      }
      return;
    }
    if (key.startsWith('route:')) {
      await onRouteMode(key.substring('route:'.length));
      return;
    }
    if (key.startsWith('system:')) {
      await onSystemProxyMode(key.substring('system:'.length));
      return;
    }
  }

  Menu _buildMenu() {
    return Menu(
      items: [
        MenuItem(
          key: 'show_window',
          label: '显示主窗口 / Show Window',
        ),
        MenuItem.separator(),
        MenuItem.checkbox(
          key: 'system:clear',
          label: '清除系统代理',
          checked: _state.systemProxyMode == 'clear',
        ),
        MenuItem.checkbox(
          key: 'system:auto',
          label: '自动配置系统代理',
          checked: _state.systemProxyMode == 'auto',
        ),
        MenuItem.checkbox(
          key: 'system:unchanged',
          label: '不改变系统代理',
          checked: _state.systemProxyMode == 'unchanged',
        ),
        MenuItem.checkbox(
          key: 'system:pac',
          label: 'PAC 模式',
          checked: _state.systemProxyMode == 'pac',
        ),
        MenuItem.separator(),
        MenuItem.submenu(
          label: '路由',
          submenu: Menu(
            items: [
              MenuItem.checkbox(
                key: 'route:rule',
                label: '规则模式',
                checked: _state.mode == 'rule',
              ),
              MenuItem.checkbox(
                key: 'route:global',
                label: '全局模式',
                checked: _state.mode == 'global',
              ),
              MenuItem.checkbox(
                key: 'route:direct',
                label: '直连模式',
                checked: _state.mode == 'direct',
              ),
            ],
          ),
        ),
        MenuItem.submenu(
          label: '节点',
          submenu: Menu(
            items: _profiles.isEmpty
                ? [MenuItem(label: '暂无节点', disabled: true)]
                : _profiles.map((profile) {
                    return MenuItem.checkbox(
                      key: 'profile:${profile.id}',
                      label: _profileLabel(profile),
                      checked: profile.id == _state.activeProfileId,
                    );
                  }).toList(growable: false),
          ),
        ),
        MenuItem.separator(),
        MenuItem(
          key: 'import_subscription',
          label: '从剪贴板导入分享链接',
        ),
        MenuItem(
          key: 'refresh',
          label: '刷新状态',
        ),
        MenuItem.separator(),
        MenuItem(
          key: _state.isRunning ? 'stop' : 'start',
          label: _state.isRunning ? '停止代理' : '启动代理',
        ),
        MenuItem(
          key: 'exit',
          label: '退出',
        ),
      ],
    );
  }

  Future<void> _showWindow() async {
    await windowManager.show();
    await windowManager.focus();
  }

  String _activeProfileName() {
    return _profiles
            .where((item) => item.id == _state.activeProfileId)
            .map((item) => item.name)
            .firstOrNull ??
        'no node';
  }

  String _profileLabel(Profile profile) {
    final group = profile.group.trim();
    if (group.isEmpty) {
      return profile.name;
    }
    return '[$group] ${profile.name}';
  }
}

extension _FirstOrNull<T> on Iterable<T> {
  T? get firstOrNull {
    final iterator = this.iterator;
    if (!iterator.moveNext()) {
      return null;
    }
    return iterator.current;
  }
}
