import 'package:flutter/widgets.dart';

class FeatureWidget extends StatelessWidget {
  const FeatureWidget({super.key});

  bool enabled(bool flag) {
    if (flag) {
      return true;
    } else {
      return false;
    }
  }

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
