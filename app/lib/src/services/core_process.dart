import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

class CoreProcess {
  CoreProcess({this.baseUrl = 'http://127.0.0.1:9727'});

  final String baseUrl;
  Process? _process;
  bool _startedByUs = false;

  Future<void> ensureStarted() async {
    if (!_isDesktop) {
      return;
    }
    if (await _isHealthy()) {
      return;
    }
    final binary = await _findBinary();
    if (binary == null) {
      return;
    }
    _process = await Process.start(
      binary.path,
      const [],
      mode: ProcessStartMode.detachedWithStdio,
    );
    _startedByUs = true;
    _process?.stdout.transform(utf8.decoder).listen((_) {});
    _process?.stderr.transform(utf8.decoder).listen((_) {});
    for (var i = 0; i < 25; i++) {
      if (await _isHealthy()) {
        return;
      }
      await Future<void>.delayed(const Duration(milliseconds: 200));
    }
  }

  void dispose() {
    if (_startedByUs) {
      _process?.kill();
    }
  }

  bool get _isDesktop => Platform.isWindows || Platform.isLinux || Platform.isMacOS;

  Future<bool> _isHealthy() async {
    try {
      final response = await http
          .get(Uri.parse('$baseUrl/health'))
          .timeout(const Duration(milliseconds: 600));
      return response.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<File?> _findBinary() async {
    final env = Platform.environment['KRAYN_CORE'];
    if (env != null && env.isNotEmpty) {
      final file = File(env);
      if (await file.exists()) {
        return file;
      }
    }

    final name = Platform.isWindows ? 'krayn-core.exe' : 'krayn-core';
    final roots = <Directory>[
      File(Platform.resolvedExecutable).parent,
      Directory.current,
      Directory('${Directory.current.path}${Platform.pathSeparator}core'),
      Directory(
        '${Directory.current.parent.path}${Platform.pathSeparator}core',
      ),
    ];
    for (final root in roots) {
      final file = File('${root.path}${Platform.pathSeparator}$name');
      if (await file.exists()) {
        return file;
      }
    }
    return null;
  }
}
