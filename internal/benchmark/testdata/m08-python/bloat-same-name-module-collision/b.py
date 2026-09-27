def other(value):
    return value

def _wrapper(value):
    return other(value)

def first(value):
    return _wrapper(value)

def second(value):
    return _wrapper(value)
