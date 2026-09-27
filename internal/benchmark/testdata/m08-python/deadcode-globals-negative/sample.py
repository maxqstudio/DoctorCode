def _hook():
    return 1

def load():
    return globals()["_hook"]()
