// Command ujobs serves a web GUI for an ai-job-search checkout
// (https://github.com/ai-job-search-dev/ai-job-search).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/SamuelSchutz13/uJobsGUI/internal/api"
	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
	"github.com/SamuelSchutz13/uJobsGUI/web"
)

func main() {
	root := flag.String("root", os.Getenv("AI_JOB_SEARCH_DIR"), "ai-job-search folder (or $AI_JOB_SEARCH_DIR)")
	router := flag.String("router", "chi", "HTTP framework: "+strings.Join(api.Routers, ", "))
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (keep it on loopback)")
	claude := flag.String("claude", "claude", "Claude Code executable")
	bun := flag.String("bun", "bun", "Bun executable")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "set --root to your ai-job-search folder")
		flag.Usage()
		os.Exit(2)
	}
	s, err := core.New(*root, *claude, *bun)
	if err != nil {
		log.Fatal(err)
	}
	h, err := api.New(*router, s, web.Dist())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ujobs (%s) on http://%s using %s", *router, *addr, s.Root)
	log.Fatal(http.ListenAndServe(*addr, h))
}
