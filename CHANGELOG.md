# Changelog

## EDxDC Changelog

## [v1.1.0-beta] - XX-XX-XXXX [PLANNED RELEASE]

### Added

- Installer script to be used by Inno Setup during build to facilitate installer releases

## [v1.0.0-beta] - 07-12-2025

This release takes previous work from the fork and gives it it's own repo and a new name reflecting the intent to support other types of displays in future releases.

## EDx52display Fork Changelog

## [v0.2.4] - XX-XX-XXXX

> **INFO**: _Orphaned release. Rolled into v1.0.0-beta of EDxDC_ 

### Added

- About page, accessed by right clicking system tray icon and selecting `About`
- System notification when app has successfully started

## [v0.2.3] - 07-12-2025

### Fixed

- Automatic updates are more careful about not deleting unrelated files if they're in a non-standard installation
- Temp files are in their own directory

## [v0.2.2] - 07-12-2025

### Added

- Automatic version check and update functions

> **INFO:** _This feature is being released on its own to simplify future updates for users, as development is progressing quickly_

## [v0.2.1] - 07-11-2025

### Added

- PORT pages for current and targeted star and ground ports
- FLEET CARRIER pages for current and targeted FCs

### Changed

- Several header names
- Casing of body types to capital case
- Changed all `CUR` headers to `CURR`

### Fixed

- Cargo page blanking with empty cargo hold
- Unreliable detection behavior for target and location type

### Removed

- UPX compression as Windows won't stop flagging it as malware.

## [v0.2.0] - 07-07-2025

### Added

- Jumps remaining to FSD Target page
- Arrival screen when route is complete
- Loading splashscreen (Needs improvement)
- Support for the Panther Clipper's massive cargo hold by displaying 4 digits on the cargo screen

### Changed

- Polling to OS-level notifications, faster and more efficient
- Most value formatting to be right-aligned, may change more in future releases
- Credit value formatting to include commas in the appropriate places for better readability
- Layout of most pages to be more information dense and look better (Credit to [pbxx](https://github.com/pbxx))
- String handling to be more robust and allow for easier page additions in future releases (Credit to [pbxx](https://github.com/pbxx))

### Fixed

- Target page sometimes displaying unlocalized name
- Rare commodities displaying the category of the commodity instead of the name
- Outdated commodity CSVs (May still be incomplete)

### Known Bugs

- Selecting a system from the left-side external panel results in either 0 or 16 jumps remaining being displayed. This is unfortunately a bug with ED's journaling where it's actually displaying those number as jumps remaining so this will require a fix on Frontier's end
- Arrival screen sometimes displays a few seconds after startup
- Splash screen gets stuck waiting for a destination
- Switching between Local Target and Next Jump views sometimes requires a page scroll.

## [v0.1.3] - 06-29-2025

### Added

- `-s -w` flags to strip debug info
- UPX compression
  ※ These changes result in the release executable dropping from about
  11MB to 2.4MB!!!

### Changed

- Default polling rate to 500ms

## [v0.1.2] - 06-29-2025

### Added

- Page registry and config-driven page toggling.

## [v0.1.1] - 06-29-2025

### Added

- Page registry and config-driven page toggling.
- System tray support with quit option.
- Logging to rotating files in the `logs` directory.
- Icon embedding for system tray and executable.
- Cargo page now shows "Cargo Hold Empty" when appropriate.

### Changed

- Destination page now dynamically shows local or FSD target.
- Logging format and file naming improved.

### Fixed

- Cargo page no longer shows "No cargo data" when cargo is empty.
- Fixed issues with duplicate function names and package imports.

## [v0.1.0] - 06-28-2025

### Added

- Initial fixes and journal parsing
