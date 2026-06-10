import 'package:flutter/material.dart';

class KrayNLogoMark extends StatelessWidget {
  const KrayNLogoMark({
    super.key,
    this.size = 40,
    this.active = true,
  });

  final double size;
  final bool active;

  @override
  Widget build(BuildContext context) {
    return SizedBox.square(
      dimension: size,
      child: CustomPaint(
        painter: _KrayNLogoPainter(active: active),
      ),
    );
  }
}

class _KrayNLogoPainter extends CustomPainter {
  const _KrayNLogoPainter({required this.active});

  final bool active;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final scale = size.shortestSide / 128;

    final background = Paint()
      ..shader = const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          Color(0xff08343a),
          Color(0xff006c67),
          Color(0xff1aa89c),
        ],
      ).createShader(rect);

    canvas.drawRRect(
      RRect.fromRectAndRadius(rect, Radius.circular(24 * scale)),
      background,
    );

    if (!active) {
      canvas.drawRRect(
        RRect.fromRectAndRadius(rect, Radius.circular(24 * scale)),
        Paint()..color = Colors.black.withValues(alpha: 0.24),
      );
    }

    final glow = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 7 * scale
      ..strokeCap = StrokeCap.round
      ..color = const Color(0xff73fff0).withValues(alpha: active ? 0.28 : 0.16);
    canvas.drawLine(Offset(31 * scale, 36 * scale), Offset(31 * scale, 92 * scale), glow);
    canvas.drawLine(Offset(31 * scale, 64 * scale), Offset(80 * scale, 31 * scale), glow);
    canvas.drawLine(Offset(31 * scale, 64 * scale), Offset(88 * scale, 98 * scale), glow);

    final mark = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 9 * scale
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round
      ..color = const Color(0xfff7fffb);
    canvas.drawLine(Offset(36 * scale, 30 * scale), Offset(36 * scale, 98 * scale), mark);
    canvas.drawLine(Offset(36 * scale, 64 * scale), Offset(78 * scale, 31 * scale), mark);
    canvas.drawLine(Offset(37 * scale, 64 * scale), Offset(89 * scale, 98 * scale), mark);

    final ray = Path()
      ..moveTo(76 * scale, 27 * scale)
      ..lineTo(58 * scale, 63 * scale)
      ..lineTo(80 * scale, 63 * scale)
      ..lineTo(64 * scale, 103 * scale)
      ..lineTo(101 * scale, 53 * scale)
      ..lineTo(78 * scale, 53 * scale)
      ..close();
    canvas.drawPath(ray, Paint()..color = const Color(0xffffb84d));

    final nodePaint = Paint()..color = const Color(0xff8ffff2);
    canvas.drawCircle(Offset(94 * scale, 32 * scale), 4.5 * scale, nodePaint);
    canvas.drawCircle(Offset(100 * scale, 92 * scale), 4.5 * scale, nodePaint);
    canvas.drawLine(
      Offset(94 * scale, 32 * scale),
      Offset(103 * scale, 49 * scale),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.5 * scale
        ..strokeCap = StrokeCap.round
        ..color = const Color(0xff8ffff2).withValues(alpha: 0.7),
    );
  }

  @override
  bool shouldRepaint(_KrayNLogoPainter oldDelegate) {
    return oldDelegate.active != active;
  }
}
