# Coolnet Cam

A Home Assistant add-on that logs into a Coolnet camera account, keeps the
streams alive, and serves a low-latency HLS proxy plus a web grid. It runs
entirely inside Home Assistant as a Supervisor-managed add-on — no external
server required.

## Configuration

| Option | Description |
| ------ | ----------- |
| `username` | Your Coolnet account username |
| `password` | Your Coolnet account password |

Set both on the **Configuration** tab, then **Save** and **Start**. The add-on
will refuse to start until both are set.

## Usage

1. Install and start the add-on (see the repository
   [README](https://github.com/daylioti/ha-coolnetcam) for adding the repository).
2. Watch the **Log** tab for `Logged in as ...` and one `Camera: ... channel=...`
   line per discovered camera.
3. Click **Open Web UI** for the live grid. The channel IDs are also listed at
   `http://<HA-IP>:8090/api/cameras`.

## Add the cameras to Home Assistant (Generic Camera, one per stream)

**Easiest:** open the Web UI and click **⚙ Add to Home Assistant** (or visit
`http://<HA-IP>:8090/setup`). It lists every camera with a **Copy** button for the
exact **Stream Source URL** to paste — no hand-building URLs. The `/api/cameras`
JSON also includes a ready-to-use `streamSource` field for each camera.

Then, for each camera:

- **Settings → Devices & Services → Add Integration → Generic Camera**
- **Stream Source URL:** paste from the Setup page
  (`http://<HA-IP>:8090/stream/<channelId>/stream.m3u8`)
- **Still Image URL:** leave blank (derived from stream)
- Name it (e.g. `Coolnet Front`)

Use the Home Assistant host IP (e.g. `http://192.168.x.x:8090/...`), **not**
`http://local-coolnetcam:.../`, so casting to a Chromecast works — the
Chromecast fetches the stream itself and isn't on the add-on's docker network.

## Low-latency dashboard view (recommended)

Home Assistant's Generic Camera cards stream through HA's `stream` integration,
which adds several seconds of buffering. For near-real-time viewing, embed the
add-on's own player instead: add a **Webpage** (iframe) card — a full-screen
**panel** view works well — pointing at `http://<HA-IP>:8090/`. The built-in
grid uses a tuned low-latency player on playlists trimmed to the live edge, and
auto-arranges into a wall that fills the screen.

## Updating

After changing the add-on source, bump `version:` in `config.yaml` and push to
the repository. Home Assistant will offer an **Update** on the add-on card.

## Network

The add-on exposes port `8090/tcp` for the HLS proxy and web UI. The upstream
camera host is `https://video.coolnet.com.ua`.
