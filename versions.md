# Versions

## v0.2.0 (2026-03-10)

- Added version resolver:
  - `latest` resolves to the highest available (for install) or installed (for use)
  - prefixes like `1` or `1.22` resolve to highest matching `1.x.y` or `1.22.x`
  
## v0.1.0 (2025-03-08)

- Initial release.
- Added runtimes:
  - go
  - node
  - python
- Added global commands:
  - version/v => list versions
  - update => upgrade runtimez version
  - purge => purge all data (apart from logs)
- Added runtime commands:
  - i/install *version* => install a given *version*
  - u/uninstall *version* => uninstall a given *version*
  - use *version* => to set *version* as active
  - ls => list available/installed *versions*
  - purge => to remove runtime data

