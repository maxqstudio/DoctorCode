import sys

def _hook():
    return 1

def load():
    return getattr(sys.modules[__name__], "_hook")()
