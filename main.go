package main

import (
	"fmt"
	"os"
)

// Build metadata. Overridden via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var subcommands = map[string]bool{
	"servers": true, "info": true, "search": true, "formats": true,
	"bodies": true, "service-bodies": true, "changes": true,
	"keys": true, "field-keys": true, "values": true, "field-values": true,
	"naws": true, "coverage": true,
	"help": true, "-h": true, "--help": true,
	"version": true, "-v": true, "--version": true,
}

// extractCommand finds the first arg that is a known subcommand and returns
// (cmd, remaining args with cmd removed). Allows "bmlt -s X search ..." as well
// as "bmlt search -s X ...".
func extractCommand(args []string) (string, []string) {
	for i, a := range args {
		if subcommands[a] {
			return a, append(append([]string{}, args[:i]...), args[i+1:]...)
		}
	}
	return "", args
}

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	cmd, args := extractCommand(os.Args[1:])
	if cmd == "" {
		fmt.Fprintf(os.Stderr, "bmlt: no command found in: %v\n\n", os.Args[1:])
		printUsage(os.Stderr)
		os.Exit(2)
	}

	switch cmd {
	case "servers":
		runServers(args)
	case "info":
		runInfo(args)
	case "search":
		runSearch(args)
	case "formats":
		runFormats(args)
	case "bodies", "service-bodies":
		runBodies(args)
	case "changes":
		runChanges(args)
	case "keys", "field-keys":
		runFieldKeys(args)
	case "values", "field-values":
		runFieldValues(args)
	case "naws":
		runNAWS(args)
	case "coverage":
		runCoverage(args)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
	case "version", "-v", "--version":
		fmt.Printf("bmlt %s (commit %s, built %s)\n", version, commit, date)
	default:
		fmt.Fprintf(os.Stderr, "bmlt: unknown command %q\n\n", cmd)
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `bmlt — CLI for the BMLT Semantic API

USAGE
  bmlt <command> [flags]

DISCOVERY
  servers [query]            List known BMLT root servers (filter by substring)

SERVER METADATA  (require -s/--server)
  info                       Server version, features, aggregator status
  coverage                   Bounding box of all meetings
  keys                       Available meeting field keys
  values -k <field>          Distinct values for a field
  formats                    Meeting format definitions
  bodies                     Service bodies (regions/areas) tree

DATA
  search [flags]             Find meetings (the workhorse)
  changes [flags]            Meeting change history
  naws -b <service-body-id>  NAWS-format CSV export

GLOBAL FLAGS
  -s, --server <url|name>    Root server URL, or fuzzy name from "bmlt servers"
                             (env: BMLT_SERVER)
      --json                 Output raw JSON instead of human format
      --tsml                 Output TSML JSON (search only)
      --url                  Print the request URL and exit (don't fetch)
      --timeout <sec>        HTTP timeout (default 30)

EXAMPLES
  bmlt servers ohio
  bmlt -s https://bmlt.sezf.org/main_server/ info
  bmlt -s "Ohio Region" bodies
  bmlt -s "Ohio Region" search --weekdays mon,wed --venue-types virtual
  bmlt -s https://bmlt.sezf.org/main_server/ search \
       --near 39.96,-83.00 --radius-km 5 --fields meeting_name,start_time

Run "bmlt <command> -h" for command-specific flags.
`)
}
