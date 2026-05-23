# Changelog

## 1.1.0

- New **Setup** page (`/setup`, or the "⚙ Add to Home Assistant" link in the
  Web UI) listing every camera with a one-click copy button for the exact
  Generic Camera **Stream Source URL**.
- `/api/cameras` now returns absolute, copy-paste-ready `streamSource` and
  `suggestedName` fields.

## 1.0.0

- Initial release.
- Coolnet account login with automatic session keepalive.
- Low-latency HLS proxy that trims playlists to the live edge for instant start.
- Built-in web grid UI (click a tile for fullscreen).
- Periodic camera re-discovery and automatic re-login.
