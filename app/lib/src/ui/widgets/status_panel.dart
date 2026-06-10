import 'package:flutter/material.dart';

import '../../models/profile.dart';
import '../../models/runtime_state.dart';
import '../formatters.dart';

class StatusPanel extends StatelessWidget {
  const StatusPanel({
    super.key,
    required this.state,
    required this.activeProfile,
    required this.busy,
    required this.onStartStop,
  });

  final RuntimeState state;
  final Profile? activeProfile;
  final bool busy;
  final VoidCallback onStartStop;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = state.isRunning ? theme.colorScheme.primary : theme.colorScheme.outline;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: LayoutBuilder(
          builder: (context, constraints) {
            final compact = constraints.maxWidth < 720;
            final header = Row(
              children: [
                Container(
                  width: 56,
                  height: 56,
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(
                    state.isRunning ? Icons.power_settings_new : Icons.power_off,
                    color: color,
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        activeProfile?.name ?? 'No active node',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.titleMedium,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '${state.status.toUpperCase()}  SOCKS ${state.socksAddress.isEmpty ? '-' : state.socksAddress}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                FilledButton.icon(
                  onPressed: busy ? null : onStartStop,
                  icon: Icon(state.isRunning ? Icons.stop : Icons.play_arrow),
                  label: Text(state.isRunning ? 'Stop' : 'Start'),
                ),
              ],
            );
            final metrics = Wrap(
              spacing: 12,
              runSpacing: 10,
              children: [
                _Metric(label: 'Up', value: formatBytes(state.uploadedBytes)),
                _Metric(label: 'Down', value: formatBytes(state.downloadedBytes)),
                _Metric(label: 'Conn', value: '${state.activeConnections}'),
              ],
            );
            if (compact) {
              return Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  header,
                  const SizedBox(height: 14),
                  metrics,
                ],
              );
            }
            return Row(
              children: [
                Expanded(child: header),
                const SizedBox(width: 16),
                metrics,
              ],
            );
          },
        ),
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ConstrainedBox(
      constraints: const BoxConstraints(minWidth: 72),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.titleSmall,
          ),
        ],
      ),
    );
  }
}
