# CTS — Compatibility Test Suite

`cts/` is the automated enforcer of [`contract/COMPATIBILITY.md`](../contract/COMPATIBILITY.md).
The spec *defines* what an Autonomous-compatible device must do; this suite *verifies* it.
Passing `cts` is what lets a device use the Autonomous-compatible mark and run the skill
catalog — the mechanism that keeps a hundreds-of-device ecosystem coherent (Android's CTS).

## Run

```bash
python3 -m unittest discover -s cts -v
```

It validates every real device under `devices/` against the MUST rules: schema, the `system`
capability, a primary sense/output, a known capability vocabulary, and a `SAFETY.md` for any
safety-class capability. It runs with no hardware — it checks contracts, not silicon.

## Scope today / next

- **Today:** static validation of `DEVICE.md` against the spec.
- **Next:** runtime conformance against a live device (capability endpoints answer, the
  deterministic stop halts within budget, the standard API envelope), gated behind a device
  target — the equivalent of Android's VTS.
