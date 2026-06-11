import 'package:flutter/material.dart';

import 'src/services/core_api.dart';
import 'src/services/core_process.dart';
import 'src/ui/app.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final process = CoreProcess();
  runApp(KrayNApp(api: CoreApi(), process: process));
}
