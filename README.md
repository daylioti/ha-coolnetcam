<p align="center">
  <img src="coolnetcam/logo.png" alt="Coolnet Cam" width="440">
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://github.com/daylioti/ha-coolnetcam/actions/workflows/lint.yaml"><img src="https://github.com/daylioti/ha-coolnetcam/actions/workflows/lint.yaml/badge.svg" alt="Lint"></a>
  <img src="https://img.shields.io/badge/arch-aarch64%20%7C%20amd64-informational" alt="Architectures">
</p>

# Coolnet Cam — Home Assistant add-on

A Home Assistant **add-on** that logs into a [Coolnet](https://video.coolnet.com.ua)
camera account, keeps the streams alive, and serves a **low-latency HLS proxy**
plus a built-in web grid. Point Home Assistant's Generic Camera integration — or
a Chromecast — at the proxy and watch your cameras with near-real-time latency.

> **Not a HACS integration.** HACS installs custom integrations, Lovelace cards
> and themes — not add-ons. This is a Docker-based add-on, so it's installed by
> adding this repository to the **Add-on Store**, not through HACS.

## Features

- 🔑 **Hands-off login & keepalive** — signs in and pings each stream so it never
  goes idle.
- ⚡ **Low latency** — playlists are trimmed to the live edge for an instant start.
- 🧩 **Plays nicely with HA** — expose each camera via the Generic Camera
  integration, then use it on dashboards, automations, and Chromecast.
- 🖥️ **Built-in grid UI** — open the add-on Web UI to see every camera at once.
- 🔁 **Self-healing** — periodic re-discovery and automatic re-login.

## Install

[![Open your Home Assistant instance and show the add add-on repository dialog with a specific repository URL pre-filled.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fdaylioti%2Fha-coolnetcam)

1. Click the button above, **or** in Home Assistant go to
   **Settings → Add-ons → Add-on Store → ⋮ (top right) → Repositories** and add:

   ```
   https://github.com/daylioti/ha-coolnetcam
   ```

2. The **Coolnet Cam** card appears under a new "Coolnet Cam Add-ons" section.
   Open it → **Install**.
3. Open the **Configuration** tab, enter your Coolnet `username` / `password`, **Save**.
4. **Start** the add-on. Enable **Start on boot** and **Watchdog**.
5. Check the **Log** tab for `Logged in as ...` and `Camera: ... channel=...`.

Full add-on documentation: [`coolnetcam/DOCS.md`](coolnetcam/DOCS.md).

## How it works

```
Coolnet portal ──login/keepalive──▶  Coolnet Cam add-on  ──HLS (trimmed)──▶  HA / browser / Chromecast
   (HLS source)                      (Go proxy on :8090)
```

The add-on holds an authenticated session, periodically pings the keepalive
endpoint for every discovered camera, and re-serves each HLS playlist trimmed to
the most recent segments so players start at the live edge.

## Add-ons in this repository

| Add-on | Description |
| ------ | ----------- |
| [Coolnet Cam](coolnetcam/) | Coolnet camera login + keepalive + low-latency HLS proxy |

## License

[MIT](LICENSE) © daylioti
