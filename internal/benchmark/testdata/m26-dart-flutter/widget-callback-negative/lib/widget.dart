class DemoState {
  bool mounted = true;

  void update() {
    if (mounted) {
      setState(() {});
    }
  }

  void setState(void Function() callback) => callback();
}
