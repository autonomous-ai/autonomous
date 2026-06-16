class Omit:
    """Sentinel for "field not provided" in partial-update / config messages.

    Used instead of ``None`` because ``None`` is a meaningful value for some
    fields (e.g. "clear the whitelist"), so we need a third state distinct from
    both a real value and an explicit null. ``omit`` is the shared singleton;
    it is falsy so callers can write ``if not field:`` to mean "left unset".
    """

    def __bool__(self):
        return False


omit = Omit()
