import 'package:intl/intl.dart';

String formatBytes(int bytes) {
  if (bytes < 1024) {
    return '${bytes}B';
  }
  final units = ['KB', 'MB', 'GB', 'TB'];
  double value = bytes / 1024;
  var index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index++;
  }
  return '${NumberFormat('0.0').format(value)}${units[index]}';
}

