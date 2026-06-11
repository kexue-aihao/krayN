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
    final seen = <String>{};
    final locales = <Locale>[];

    void add(Locale locale) {
      if (seen.add(locale.toLanguageTag())) {
        locales.add(locale);
      }
    }

    add(defaultLocale);
    add(const Locale('zh', 'TW'));
    add(const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'));
    for (final languageCode in kMaterialSupportedLanguages) {
      add(Locale(languageCode));
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
  String get importSubscription => _text('importSubscription');
  String get subscriptionUrl => _text('subscriptionUrl');
  String get subscriptionUrlHint => _text('subscriptionUrlHint');
  String get subscriptionImportHelp => _text('subscriptionImportHelp');
  String get importAction => _text('importAction');
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
  String get invalidSubscriptionUrl => _text('invalidSubscriptionUrl');
  String get subscriptionRequestFailed => _text('subscriptionRequestFailed');
  String get subscriptionRequestTimeout => _text('subscriptionRequestTimeout');
  String get emptySubscription => _text('emptySubscription');
  String get noSubscriptionProfiles => _text('noSubscriptionProfiles');
  String get unsupportedSubscriptionFormat =>
      _text('unsupportedSubscriptionFormat');
  String get invalidSubscriptionProfile => _text('invalidSubscriptionProfile');
  String get clipboardEmpty => _text('clipboardEmpty');
  String get systemProxyApplyFailed => _text('systemProxyApplyFailed');
  String get diagnostics => _text('diagnostics');
  String get runDiagnostics => _text('runDiagnostics');
  String get latencyTarget => _text('latencyTarget');
  String get speedTarget => _text('speedTarget');
  String get customUrl => _text('customUrl');
  String get rttLatency => _text('rttLatency');
  String get httpsLatency => _text('httpsLatency');
  String get speedTest => _text('speedTest');
  String get egressIp => _text('egressIp');
  String get asn => _text('asn');
  String get ipPurity => _text('ipPurity');
  String get openPurity => _text('openPurity');
  String get resolvedIps => _text('resolvedIps');
  String get diagnosticErrors => _text('diagnosticErrors');
  String get resolverSettings => _text('resolverSettings');
  String get resolverType => _text('resolverType');
  String get resolverPreset => _text('resolverPreset');
  String get resolverAddress => _text('resolverAddress');
  String get resolverSystem => _text('resolverSystem');
  String get resolverDns => _text('resolverDns');
  String get resolverDoh => _text('resolverDoh');
  String get resolverCustom => _text('resolverCustom');
  String get saveResolver => _text('saveResolver');
  String get noDiagnosticResult => _text('noDiagnosticResult');
  String get testing => _text('testing');
  String get copied => _text('copied');
  String get unavailable => _text('unavailable');

  String subscriptionImported(int count) {
    return _text('subscriptionImported').replaceAll('{count}', '$count');
  }

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
  'importSubscription': '添加订阅',
  'subscriptionUrl': '订阅链接',
  'subscriptionUrlHint': '粘贴以 http:// 或 https:// 开头的订阅链接',
  'subscriptionImportHelp': '支持 krayN 原生订阅配置和 base64 包裹的配置。',
  'importAction': '导入',
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
  'invalidSubscriptionUrl': '请输入有效的订阅链接',
  'subscriptionRequestFailed': '订阅拉取失败，请检查链接或网络',
  'subscriptionRequestTimeout': '订阅拉取超时，请稍后重试',
  'emptySubscription': '订阅内容为空',
  'noSubscriptionProfiles': '订阅中没有可导入的节点',
  'unsupportedSubscriptionFormat': '暂不支持该订阅格式',
  'invalidSubscriptionProfile': '订阅中的节点字段不完整',
  'clipboardEmpty': '剪贴板为空',
  'systemProxyApplyFailed': '系统代理设置失败',
  'diagnostics': '节点诊断',
  'runDiagnostics': '开始测试',
  'latencyTarget': '延迟测试地址',
  'speedTarget': '测速地址',
  'customUrl': '自定义地址',
  'rttLatency': 'RTT 延迟',
  'httpsLatency': 'HTTPS 延迟',
  'speedTest': '下载测速',
  'egressIp': '出口 IP',
  'asn': 'ASN',
  'ipPurity': 'IP 纯净度',
  'openPurity': '打开 Ping0',
  'resolvedIps': '解析 IP',
  'diagnosticErrors': '诊断提示',
  'resolverSettings': '节点解析',
  'resolverType': '解析方式',
  'resolverPreset': '预设',
  'resolverAddress': '解析地址',
  'resolverSystem': '系统默认',
  'resolverDns': 'DNS',
  'resolverDoh': 'DoH',
  'resolverCustom': '自定义',
  'saveResolver': '保存解析',
  'noDiagnosticResult': '尚未测试',
  'testing': '测试中',
  'copied': '已复制',
  'unavailable': '不可用',
  'subscriptionImported': '已导入 {count} 个节点',
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
  'importSubscription': 'Add subscription',
  'subscriptionUrl': 'Subscription URL',
  'subscriptionUrlHint':
      'Paste a subscription URL starting with http:// or https://',
  'subscriptionImportHelp':
      'Supports native krayN subscription configs and base64-wrapped configs.',
  'importAction': 'Import',
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
  'invalidSubscriptionUrl': 'Enter a valid subscription URL',
  'subscriptionRequestFailed':
      'Could not fetch the subscription. Check the link or network.',
  'subscriptionRequestTimeout':
      'The subscription request timed out. Try again later.',
  'emptySubscription': 'The subscription is empty',
  'noSubscriptionProfiles':
      'No importable nodes were found in this subscription',
  'unsupportedSubscriptionFormat':
      'This subscription format is not supported yet',
  'invalidSubscriptionProfile':
      'A node in the subscription is missing required fields',
  'clipboardEmpty': 'Clipboard is empty',
  'systemProxyApplyFailed': 'Could not update the system proxy',
  'diagnostics': 'Node Diagnostics',
  'runDiagnostics': 'Run Test',
  'latencyTarget': 'Latency URL',
  'speedTarget': 'Speed URL',
  'customUrl': 'Custom URL',
  'rttLatency': 'RTT latency',
  'httpsLatency': 'HTTPS latency',
  'speedTest': 'Download speed',
  'egressIp': 'Egress IP',
  'asn': 'ASN',
  'ipPurity': 'IP purity',
  'openPurity': 'Open Ping0',
  'resolvedIps': 'Resolved IPs',
  'diagnosticErrors': 'Diagnostic notes',
  'resolverSettings': 'Node DNS',
  'resolverType': 'Resolver',
  'resolverPreset': 'Preset',
  'resolverAddress': 'Resolver address',
  'resolverSystem': 'System',
  'resolverDns': 'DNS',
  'resolverDoh': 'DoH',
  'resolverCustom': 'Custom',
  'saveResolver': 'Save DNS',
  'noDiagnosticResult': 'Not tested yet',
  'testing': 'Testing',
  'copied': 'Copied',
  'unavailable': 'Unavailable',
  'subscriptionImported': 'Imported {count} nodes',
  'status_running': 'Running',
  'status_starting': 'Starting',
  'status_stopped': 'Stopped',
  'status_stopping': 'Stopping',
  'status_error': 'Error',
};

const _localizedValues = {'zh': _zhValues, 'en': _enValues};
