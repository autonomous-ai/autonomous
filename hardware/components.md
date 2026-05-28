# Components

Running list of parts that go into one lamp. Models / links filled in as we lock things down.

> **Optional parts**: rows marked **Optional** in the table below are **not required** for a functional build. They are decorative, cosmetic, or quality-of-life additions and can be skipped without affecting the lamp's behaviour. They are listed here so builders know what's available, not what's mandatory.

## Required

| Part | Model | Notes |
|---|---|---|
| Mic 1 (voice) | USB dual mic HK JZMIC v1.0 | to be updated |
| Camera | USB ZV A016 V4 | FOV 78° |
| Speaker 3W x2 | | |
| Speaker amplifier | PAM8610 v2 | |
| OrangePi 4 Pro | 4GB or 6GB RAM | |
| Servo x5 | STS3215 ST-3215-C018 | |
| Servo control board (USB) | Waveshare Bus Servo Adapter | https://www.waveshare.com/product/modules/motors-servos/drivers/bus-servo-adapter-a.htm |
| RGB LED ring | WS2812B 32-LED ring | |
| User button | Tactile button switch 6 mm | https://www.adafruit.com/product/367 or equivalent. Merged with reset (hold 15 s to reset) |
| RJ45 extender | | extends the OrangePi 4 Pro Ethernet port |
| Wire, screw, header, USB-C female | | to be updated |
| 3D printed body | | STL / STEP files: see `cad/` |
| Ball bearing | | STL / STEP files: see `cad/` |
| Aluminium CNC parts | | holds the STS3215 servos; STL / STEP files: see `cad/` |
| Wooden CNC parts | | STL / STEP files: see `cad/` |
| Power adaptor | 12V 5A LiteOn | |
| DC-DC step-down (12V → 5V for OrangePi) | MP2482 | needs replacement — produces audible audio hiss |
| Fan 5V | Nidec | |
| Touch button x4 | TTP223 | mounted on the lamp head |

## Optional

These parts improve the build but are **not required** — the lamp works fine without them.

| Part | Model | Purpose |
|---|---|---|
| Limited edition tag | | **Decorative.** Units 1–100, CNC aluminium or laser engraving on leather. Pure cosmetic — skip for non-numbered builds. |
| Steel plate (dead weight) | 5 mm thick steel disc, ~3 kg | **Stability.** Ballast for the lamp base. Skip if your base is already heavy enough not to tip. |
| Silicon pad | | **Anti-slip.** Stuck to the bottom of the lamp base to reduce slipping on smooth surfaces. Skip if surface friction is fine without it. |
