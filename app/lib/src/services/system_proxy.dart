import 'dart:io';

import 'package:flutter/foundation.dart';

class SystemProxy {
  static bool get isSupportedDesktop {
    return !kIsWeb &&
        (Platform.isWindows || Platform.isMacOS || Platform.isLinux);
  }

  Future<void> apply({
    required String mode,
    required String socksAddress,
    required String pacUrl,
  }) async {
    if (!isSupportedDesktop || mode == 'unchanged') {
      return;
    }
    if (mode == 'clear') {
      await clear();
      return;
    }
    if (mode == 'pac') {
      await _applyPac(pacUrl);
      return;
    }
    await _applySocks(socksAddress);
  }

  Future<void> clear() async {
    if (!isSupportedDesktop) {
      return;
    }
    if (Platform.isWindows) {
      await _run('reg', [
        'add',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
        '/v',
        'ProxyEnable',
        '/t',
        'REG_DWORD',
        '/d',
        '0',
        '/f',
      ]);
      await _run(
          'reg',
          [
            'delete',
            r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
            '/v',
            'ProxyServer',
            '/f',
          ],
          allowFailure: true);
      await _run(
          'reg',
          [
            'delete',
            r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
            '/v',
            'AutoConfigURL',
            '/f',
          ],
          allowFailure: true);
      await _refreshWindowsProxy();
      return;
    }
    if (Platform.isMacOS) {
      for (final service in await _macNetworkServices()) {
        await _run('networksetup', ['-setwebproxystate', service, 'off']);
        await _run('networksetup', ['-setsecurewebproxystate', service, 'off']);
        await _run(
            'networksetup', ['-setsocksfirewallproxystate', service, 'off']);
        await _run('networksetup', ['-setautoproxystate', service, 'off']);
      }
      return;
    }
    if (Platform.isLinux) {
      await _run(
          'gsettings', ['set', 'org.gnome.system.proxy', 'mode', 'none']);
    }
  }

  Future<void> _applySocks(String socksAddress) async {
    final endpoint = _splitHostPort(socksAddress);
    if (Platform.isWindows) {
      await _run('reg', [
        'add',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
        '/v',
        'ProxyEnable',
        '/t',
        'REG_DWORD',
        '/d',
        '1',
        '/f',
      ]);
      await _run('reg', [
        'add',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
        '/v',
        'ProxyServer',
        '/t',
        'REG_SZ',
        '/d',
        'http=${endpoint.host}:${endpoint.port};https=${endpoint.host}:${endpoint.port};socks=${endpoint.host}:${endpoint.port}',
        '/f',
      ]);
      await _run(
          'reg',
          [
            'delete',
            r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
            '/v',
            'AutoConfigURL',
            '/f',
          ],
          allowFailure: true);
      await _refreshWindowsProxy();
      return;
    }
    if (Platform.isMacOS) {
      for (final service in await _macNetworkServices()) {
        await _run('networksetup', [
          '-setwebproxy',
          service,
          endpoint.host,
          endpoint.port,
        ]);
        await _run('networksetup', ['-setwebproxystate', service, 'on']);
        await _run('networksetup', [
          '-setsecurewebproxy',
          service,
          endpoint.host,
          endpoint.port,
        ]);
        await _run('networksetup', ['-setsecurewebproxystate', service, 'on']);
        await _run('networksetup', [
          '-setsocksfirewallproxy',
          service,
          endpoint.host,
          endpoint.port,
        ]);
        await _run(
            'networksetup', ['-setsocksfirewallproxystate', service, 'on']);
        await _run('networksetup', ['-setautoproxystate', service, 'off']);
      }
      return;
    }
    if (Platform.isLinux) {
      await _run(
          'gsettings', ['set', 'org.gnome.system.proxy', 'mode', 'manual']);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.socks', 'host', endpoint.host]);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.socks', 'port', endpoint.port]);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.http', 'host', endpoint.host]);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.http', 'port', endpoint.port]);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.https', 'host', endpoint.host]);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy.https', 'port', endpoint.port]);
    }
  }

  Future<void> _applyPac(String pacUrl) async {
    if (Platform.isWindows) {
      await _run('reg', [
        'add',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
        '/v',
        'ProxyEnable',
        '/t',
        'REG_DWORD',
        '/d',
        '0',
        '/f',
      ]);
      await _run('reg', [
        'add',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings',
        '/v',
        'AutoConfigURL',
        '/t',
        'REG_SZ',
        '/d',
        pacUrl,
        '/f',
      ]);
      await _refreshWindowsProxy();
      return;
    }
    if (Platform.isMacOS) {
      for (final service in await _macNetworkServices()) {
        await _run('networksetup', ['-setautoproxyurl', service, pacUrl]);
        await _run('networksetup', ['-setautoproxystate', service, 'on']);
        await _run('networksetup', ['-setwebproxystate', service, 'off']);
        await _run('networksetup', ['-setsecurewebproxystate', service, 'off']);
        await _run(
            'networksetup', ['-setsocksfirewallproxystate', service, 'off']);
      }
      return;
    }
    if (Platform.isLinux) {
      await _run(
          'gsettings', ['set', 'org.gnome.system.proxy', 'mode', 'auto']);
      await _run('gsettings',
          ['set', 'org.gnome.system.proxy', 'autoconfig-url', pacUrl]);
    }
  }

  Future<List<String>> _macNetworkServices() async {
    final result = await _run('networksetup', ['-listallnetworkservices']);
    return result.stdout
        .toString()
        .split('\n')
        .map((line) => line.trim())
        .where((line) => line.isNotEmpty && !line.startsWith('An asterisk'))
        .toList(growable: false);
  }

  Future<void> _refreshWindowsProxy() async {
    await _run('powershell.exe', [
      '-NoProfile',
      '-ExecutionPolicy',
      'Bypass',
      '-Command',
      r'''
$signature = @"
[DllImport("wininet.dll", SetLastError = true)]
public static extern bool InternetSetOption(IntPtr hInternet, int dwOption, IntPtr lpBuffer, int dwBufferLength);
"@
$type = Add-Type -MemberDefinition $signature -Name WinInetRefresh -Namespace KrayN -PassThru
$type::InternetSetOption([IntPtr]::Zero, 39, [IntPtr]::Zero, 0) | Out-Null
$type::InternetSetOption([IntPtr]::Zero, 37, [IntPtr]::Zero, 0) | Out-Null
''',
    ]);
  }

  Future<ProcessResult> _run(
    String executable,
    List<String> arguments, {
    bool allowFailure = false,
  }) async {
    final result = await Process.run(executable, arguments, runInShell: false);
    if (!allowFailure && result.exitCode != 0) {
      final error = result.stderr.toString().trim();
      throw SystemProxyException(
        error.isEmpty ? '$executable exited with ${result.exitCode}' : error,
      );
    }
    return result;
  }

  _ProxyEndpoint _splitHostPort(String value) {
    final uri = Uri.tryParse('socks://$value');
    final host = uri?.host.isNotEmpty == true ? uri!.host : '127.0.0.1';
    final port = uri?.hasPort == true ? uri!.port : 7890;
    return _ProxyEndpoint(host, '$port');
  }
}

class SystemProxyException implements Exception {
  const SystemProxyException(this.message);

  final String message;

  @override
  String toString() => message;
}

class _ProxyEndpoint {
  const _ProxyEndpoint(this.host, this.port);

  final String host;
  final String port;
}
