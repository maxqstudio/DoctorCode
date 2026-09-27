def target(value):
    return value

def _wrapper(value):
    return target(value)

def first(value):
    return _wrapper(value)

def second(value):
    return _wrapper(value)
