<h1 align="center">glovo-cli</h1>

<p align="center">
  <a href="go.mod"><img src="https://img.shields.io/badge/GO-1.26%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/LICENSE-MIT-5C9E31?style=for-the-badge" alt="License"></a>
  <img src="https://img.shields.io/badge/SINGLE%20BINARY-no%20runtime%20deps-brightgreen?style=for-the-badge" alt="Single binary">
  <a href="https://rebelcode.com"><img src="https://img.shields.io/badge/BUILT%20BY-REBELCODE-8A2BE2?style=for-the-badge" alt="Built by RebelCode"></a>
</p>

<p align="center"><strong>Unofficial CLI for Glovo. Find restaurants, browse menus, and fill your own cart from the terminal.</strong></p>

---

## Overview

`glovo` is a single Go binary that talks to the same endpoints glovoapp.com uses. It's not a scraper of arbitrary pages — `search` and `cart`/`order` call Glovo's authenticated store API with your own account, and `menu` reads the server-rendered store page (no account needed). Every command supports `--json` (data to stdout, logs to stderr) and `--toon` (fewer tokens, for agents).

## Commands

```
glovo — unofficial CLI for Glovo food delivery

USAGE:
  glovo <command> [flags]

COMMANDS:
  search <query...>         find restaurants near your delivery address
  menu <store-slug|id>      list a restaurant's categories and items
  login [--email a|--token|--access-token]  authenticate with your OWN account
  cart get <store-id>       show your basket for a store
  cart add <store-id> <product-id> [qty]
  cart set <store-id> <product-id> <qty>   (0 removes)
  cart clear <store-id>
  order <store-id>          dry-run: restaurant, items, fees, total, ETA
  order <store-id> --confirm  place the order (requires captured checkout)

LOCATION FLAGS (search, menu, cart):
  --lat, --lng              delivery coordinates (or GLOVO_LAT/GLOVO_LNG)
  --city, --country         city/country codes; resolved from the coordinates
  --city-slug               city segment of store URLs, e.g. lisboa

COMMON FLAGS (after the command):
  --json                    emit raw JSON (data → stdout, logs → stderr)
  --toon                    emit TOON (fewer tokens; for agents)

ENV:
  GLOVO_CONFIG_DIR          override ~/.glovo (session, device and city cache)
```

Every command needs to know where you are ordering from: pass `--lat`/`--lng`, set `GLOVO_LAT`/`GLOVO_LNG`, or accept the Barcelona-centre fallback. The city and country codes, and the city segment of Glovo's store URLs, are looked up from those coordinates and cached, so `--city`, `--country` and `--city-slug` are only there for when you want to override the lookup.

`search` calls Glovo's authenticated store-search API and needs `glovo login` first. `menu <store-slug>` needs no account.

`search --exclude "domino,pizza hut"` hides stores by name, for the places you never want to see. Keep a standing list in `~/.glovo/exclude.txt`, one name per line, and pass `--exclude none` for a search that shows everything. Glovo publishes nothing that marks a store as a chain or a ghost kitchen, so the list is yours to keep. Whatever is hidden is reported on stderr, so a short result list is never quietly a filtered one.

## Auth

`glovo login` supports three paths:

- `--email you@example.com` — logs in with your email and password. The password is read from a hidden prompt (or piped: `pbpaste | glovo login --email you@example.com`), sent once, and never stored. Glovo verifies any device it hasn't seen before, so the first login on a machine texts you a code and the CLI prompts for it. This is the simplest terminal login.
- `--token` — paste your `glovo_refresh_token` from a logged-in browser's Local Storage. No code needed, and refresh tokens are long-lived.
- `--access-token` — paste your current access token (from a logged-in browser: DevTools → Application → Cookies → `glovo_auth_info`). Good for about an hour and can't be refreshed, so prefer one of the others.

Sessions are cached in `~/.glovo/auth.json` and refreshed automatically when the access token expires; `~/.glovo/device.json` holds the device registration that Glovo ties the verification to, so you only get texted a code once per machine. Override the directory with `GLOVO_CONFIG_DIR`. The CLI never stores your password. On a `401`, set `GLOVO_DEBUG=1` to see the server's response.

## Cart

`cart get|add|set|clear` fills your own basket through Glovo's authenticated API:

```bash
glovo login --email you@example.com
glovo cart add na-pizza-lis 41745203922 2   # add 2× a product to a store's basket
glovo cart get na-pizza-lis
glovo cart set na-pizza-lis 41745203922 0   # remove it
glovo cart clear na-pizza-lis
```

The CLI fills the cart. It never places an order on its own.

## Status: ordering

`glovo order <store-id>` is a dry-run summary (items, fees, total, ETA) built from your live basket — no request is sent. `glovo order <store-id> --confirm` is gated behind an explicit flag, but currently returns a "checkout not yet implemented" error: the real order-placement request hasn't been captured from a live checkout yet, so `--confirm` charges nothing. Browsing, login, and cart-filling all work end to end.

## Install

```bash
git clone https://github.com/jgalea/glovo-cli.git
cd glovo-cli
go build -o glovo ./cmd/glovo
./glovo version
```

## License

MIT. See [LICENSE](LICENSE).

Author: Jean Galea. Unofficial project, not affiliated with or endorsed by Glovo. It talks to the same endpoints glovoapp.com uses in your browser — use it at a sane request rate.
