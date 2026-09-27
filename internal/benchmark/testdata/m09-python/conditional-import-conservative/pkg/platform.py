import sys

if sys.platform == "win32":
    from .win_impl import _helper
else:
    from .linux_impl import _helper

def run():
    return _helper()
