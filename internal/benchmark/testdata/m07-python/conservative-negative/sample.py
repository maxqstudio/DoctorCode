api_token = "your_token_here"

def public_unused():
    return 1

def register(fn):
    return fn

@register
def _hook():
    return 1

def ready():
    return True

def logic():
    if ready():
        return 1
    elif ready():
        return 2
    return 3

def same(value):
    if value:
        return True
    else:
        return True

def target(value):
    return value

def _wrapper(value):
    return target(value)

def use_one(value):
    return _wrapper(value)

def use_two(value):
    return _wrapper(value)
