"""Simple button state viewer for OrangePi 4 Pro.

Wiring: button between header pin 11 (PL9) and any GND.
Run:    sudo /opt/lelamp/.venv/bin/python ~/test_button.py
"""

import sys
import time

import lgpio

CHIP = 1     # /dev/gpiochip1
LINE = 9     # PL9 / header pin 11

h = lgpio.gpiochip_open(CHIP)
lgpio.gpio_claim_input(h, LINE, lgpio.SET_PULL_UP)

try:
    while True:
        v = lgpio.gpio_read(h, LINE)
        msg = "press on " if v == 0 else "not press"
        sys.stdout.write(f"\r{msg}")
        sys.stdout.flush()
        time.sleep(0.05)
except KeyboardInterrupt:
    print()
finally:
    lgpio.gpiochip_close(h)