import 'dart:collection';

import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

class KrayNLocalizations {
  const KrayNLocalizations(this.locale);

  final Locale locale;

  static const defaultLocale = Locale('zh', 'CN');
  static const LocalizationsDelegate<KrayNLocalizations> delegate =
      _KrayNLocalizationsDelegate();

  static final supportedLocales = _buildSupportedLocales();

  static KrayNLocalizations of(BuildContext context) {
    return Localizations.of<KrayNLocalizations>(context, KrayNLocalizations)!;
  }

  static List<Locale> _buildSupportedLocales() {
    final seen = LinkedHashSet<String>();
    final locales = <Locale>[];

    void add(Locale locale) {
      if (seen.add(locale.toLanguageTag())) {
        locales.add(locale);
      }
    }

    add(defaultLocale);
    add(const Locale('zh', 'TW'));
    add(const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'));
    for (final locale in GlobalMaterialLocalizations.supportedLocales) {
      add(locale);
    }
    return List<Locale>.unmodifiable(locales);
  }

  Map<String, String> get _values {
    return _localizedValues[locale.languageCode] ?? _zhValues;
  }

  String _text(String key) {
    return _values[key] ?? _zhValues[key] ?? _enValues[key] ?? key;
  }

  String? _rawText(String key) {
    return _values[key] ?? _zhValues[key] ?? _enValues[key];
  }

  String get appTitle => 'krayN';
  String get refresh => _text('refresh');
  String get language => _text('language');
  String get chooseLanguage => _text('chooseLanguage');
  String get close => _text('close');
  String get nodes => _text('nodes');
  String get addNode => _text('addNode');
  String get noNodes => _text('noNodes');
  String get activate => _text('activate');
  String get active => _text('active');
  String get newNode => _text('newNode');
  String get nodeSettings => _text('nodeSettings');
  String get delete => _text('delete');
  String get save => _text('save');
  String get name => _text('name');
  String get group => _text('group');
  String get transport => _text('transport');
  String get endpoint => _text('endpoint');
  String get clientId => _text('clientId');
  String get tlsServerName => _text('tlsServerName');
  String get clientSecret => _text('clientSecret');
  String get serverPublicKey => _text('serverPublicKey');
  String get paddingMin => _text('paddingMin');
  String get paddingMax => _text('paddingMax');
  String get headers => _text('headers');
  String get remark => _text('remark');
  String get skipTlsVerification => _text('skipTlsVerification');
  String get requiredField => _text('requiredField');
  String get nonNegativeNumber => _text('nonNegativeNumber');
  String get noActiveNode => _text('noActiveNode');
  String get start => _text('start');
  String get stop => _text('stop');
  String get upload => _text('upload');
  String get download => _text('download');
  String get connections => _text('connections');
  String get socks => _text('socks');

  String runtimeStatus(String status) {
    final key = 'status_${status.toLowerCase().replaceAll('-', '_')}';
    return _rawText(key) ?? status.toUpperCase();
  }
}

class _KrayNLocalizationsDelegate
    extends LocalizationsDelegate<KrayNLocalizations> {
  const _KrayNLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) => true;

  @override
  Future<KrayNLocalizations> load(Locale locale) async {
    return KrayNLocalizations(locale);
  }

  @override
  bool shouldReload(_KrayNLocalizationsDelegate old) => false;
}

const _zhValues = {
  'refresh': '刷新',
  'language': '语言',
  'chooseLanguage': '选择语言',
  'close': '关闭',
  'nodes': '节点',
  'addNode': '添加节点',
  'noNodes': '暂无节点',
  'activate': '启用',
  'active': '已启用',
  'newNode': '新建节点',
  'nodeSettings': '节点设置',
  'delete': '删除',
  'save': '保存',
  'name': '名称',
  'group': '分组',
  'transport': '传输协议',
  'endpoint': '服务器地址',
  'clientId': '客户端 ID',
  'tlsServerName': 'TLS 服务器名称',
  'clientSecret': '客户端密钥',
  'serverPublicKey': '服务器公钥',
  'paddingMin': '最小填充',
  'paddingMax': '最大填充',
  'headers': '请求头',
  'remark': '备注',
  'skipTlsVerification': '跳过 TLS 验证',
  'requiredField': '必填',
  'nonNegativeNumber': '请输入非负数字',
  'noActiveNode': '未启用节点',
  'start': '启动',
  'stop': '停止',
  'upload': '上传',
  'download': '下载',
  'connections': '连接',
  'socks': 'SOCKS',
  'status_running': '运行中',
  'status_starting': '启动中',
  'status_stopped': '已停止',
  'status_stopping': '停止中',
  'status_error': '错误',
};

const _enValues = {
  'refresh': 'Refresh',
  'language': 'Language',
  'chooseLanguage': 'Choose language',
  'close': 'Close',
  'nodes': 'Nodes',
  'addNode': 'Add node',
  'noNodes': 'No nodes yet',
  'activate': 'Activate',
  'active': 'Active',
  'newNode': 'New Node',
  'nodeSettings': 'Node Settings',
  'delete': 'Delete',
  'save': 'Save',
  'name': 'Name',
  'group': 'Group',
  'transport': 'Transport',
  'endpoint': 'Endpoint',
  'clientId': 'Client ID',
  'tlsServerName': 'TLS Server Name',
  'clientSecret': 'Client Secret',
  'serverPublicKey': 'Server Public Key',
  'paddingMin': 'Padding Min',
  'paddingMax': 'Padding Max',
  'headers': 'Headers',
  'remark': 'Remark',
  'skipTlsVerification': 'Skip TLS Verification',
  'requiredField': 'Required',
  'nonNegativeNumber': 'Use a non-negative number',
  'noActiveNode': 'No active node',
  'start': 'Start',
  'stop': 'Stop',
  'upload': 'Up',
  'download': 'Down',
  'connections': 'Conn',
  'socks': 'SOCKS',
  'status_running': 'Running',
  'status_starting': 'Starting',
  'status_stopped': 'Stopped',
  'status_stopping': 'Stopping',
  'status_error': 'Error',
};

const _localizedValues = {
  'zh': _zhValues,
  'en': _enValues,
};
