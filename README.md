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
  login [--token|--email a] authenticate with your OWN Glovo account
  cart get <store-id>       show your basket for a store
  cart add <store-id> <product-id> [qty]
  cart set <store-id> <product-id> <qty>   (0 removes)
  cart clear <store-id>
  order <store-id>          dry-run: restaurant, items, fees, total, ETA
  order <store-id> --confirm  place the order (requires captured checkout)

COMMON FLAGS (after the command):
  --json                    emit raw JSON (data → stdout, logs → stderr)
  --toon                    emit TOON (fewer tokens; for agents)

ENV:
  GLOVO_CONFIG_DIR          override ~/.glovo (token cache)
```

`search` calls Glovo's authenticated store-search API, so it needs `glovo login` first, plus delivery coordinates: `--lat`/`--lng`, or `GLOVO_LAT`/`GLOVO_LNG`, or it falls back to Barcelona centre (`--city`/`--country` default to `BCN`/`ES`). `menu <store-slug>` needs no login — it reads Glovo's public, server-rendered store page.

## Auth

`glovo login` supports two paths:

- `--token` — paste your `glovo_refresh_token` from a logged-in browser session (DevTools → Application → Local Storage). Always works, no password involved.
- `--email you@example.com` — logs in with email and password; the password is read from a hidden prompt and never stored, only the resulting tokens are.

Tokens are cached in `~/.glovo/auth.json` (override the directory with `GLOVO_CONFIG_DIR`). The CLI never stores your password.

## Cart

`cart get|add|set|clear` fills your own basket through Glovo's authenticated API:

```bash
glovo login
glovo cart add 123456 789012 2   # add 2× a product to a store's basket
glovo cart get 123456
glovo cart set 123456 789012 0   # remove it
glovo cart clear 123456
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
