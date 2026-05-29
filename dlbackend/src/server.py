"""Backward-compatible entry point. Use `python -m dlserver` instead."""

from dlserver.app import app, main

__all__ = ["app"]
if __name__ == "__main__":
    main()
