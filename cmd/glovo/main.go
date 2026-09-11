package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	args = hoistGlobalFlags(args)
	if len(args) < 1 {
		usage()
		return 2
	}
	var err error
	switch args[0] {
	case "search":
		err = cmdSearch(args[1:])
	case "menu":
		err = cmdMenu(args[1:])
	case "cart":
		err = cmdCart(args[1:])
	case "order":
		err = cmdOrder(args[1:])
	case "login":
		err = cmdLogin(args[1:])
	case "version", "--version", "-v":
		fmt.Println(versionString())
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func hoistGlobalFlags(args []string) []string {
	var lead []string
	i := 0
	for i < len(args) {
		a := args[i]
		if len(a) < 2 || a[0] != '-' {
			break
		}
		lead = append(lead, a)
		i++
	}
	if i >= len(args) {
		return args
	}
	out := make([]string, 0, len(args))
	out = append(out, args[i])
	out = append(out, args[i+1:]...)
	return append(out, lead...)
}

func versionString() string {
	if commit == "" {
		return version
	}
	if date != "" {
		return fmt.Sprintf("%s (%s, %s)", version, commit, date)
	}
	return fmt.Sprintf("%s (%s)", version, commit)
}

func usage() {
	fmt.Fprint(os.Stderr, `glovo — unofficial CLI for Glovo food delivery

USAGE:
  glovo <command> [flags]

COMMANDS:
  search <query...>         find restaurants near your delivery address
  menu <store-slug|id>      list a restaurant's categories and items
  login [--email a|--token|--access-token]  authenticate with your OWN Glovo account
                            --email logs in with your password, then prompts for
                            the verification code Glovo texts you
                            --token pastes a glovo_refresh_token from a browser
                            --access-token pastes your current access token; the
                            CLI derives your customer id from it (or pass --customer-id)
  cart get <store-slug>     show your basket for a store
  cart add <store-slug> <product-id> [qty]
  cart set <store-slug> <product-id> <qty>   (0 removes)
  cart clear <store-slug>
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

  version | help

Unofficial. Talks to the same endpoints glovoapp.com uses. Browse/menu need no
account; cart/order use your own login and never place an order without --confirm.
`)
}
