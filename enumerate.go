// +build ignore

// Enumerate unique keys from key/values found in the whois responses.
// To use: go run enumerate.go

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/domainr/whois"
	"github.com/domainr/whoistest"
	"github.com/wsxiaoys/terminal/color"
)

var (
	keys = make(map[string]string)
)

func main() {
	flag.Parse()
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main1() error {
	fns, err := whoistest.ResponseFiles()
	if err != nil {
		return err
	}
	for _, fn := range fns {
		res, err := whois.ReadMIMEFile(fn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response file %s: %s\n", fn, err)
			continue
		}
		if res.MediaType != "text/plain" {
			continue
		}
		scan(res)
	}

	sorted := make([]string, 0, len(keys))
	for k, _ := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	color.Printf("\n@{|w}%d unique keys parsed:\n", len(keys))
	for _, k := range sorted {
		color.Printf("@{|c}%- 40s  @{|.}%s\n", k, keys[k])
	}

	return nil
}

var (
	reEmptyLine   = regexp.MustCompile(`^\s*$`)
	reBareKey     = regexp.MustCompile(`^\s*([^\:,]{1,39}\S)\s*\:\s*$`)
	reKeyValue    = regexp.MustCompile(`^\s*([^\:,]{1,39}\S)\s*\:\s*(.*\S)\s*$`)
	reAltKey      = regexp.MustCompile(`^\s*\[([^\],]{1,39}\S)\]\s*$`)
	reAltKeyValue = regexp.MustCompile(`^\s*\[([^\],]{1,39}\S)\]\s*(.*\S)\s*$`)
	reBareValue   = regexp.MustCompile(`^      \s+(.*\S)\s*$`)
	reNotice      = regexp.MustCompile(strings.Join([]string{
		`^% .*$`,            // whois.de
		`^\[ .+ \]$`,        // whois.jprs.jp
		`^# .*$`,            // whois.kr
		`^>>>.+<<<$`,        // Database last updated...
		`^[^\:]+https?\://`, // Line with an URL
		`^NOTE: `,
		`^NOTICE: `,
	}, "|"))
)

func scan(res *whois.Response) {
	r, err := res.Reader()
	if err != nil {
		return
	}
	line := 0
	s := bufio.NewScanner(r)
	for s.Scan() {
		line++
		color.Printf("@{|.}% 4d  ", line)

		// Get next line
		text := s.Text()

		// Notices and empty lines
		if reEmptyLine.MatchString(text) {
			color.Printf("@{|w}EMPTY\n")
			continue
		}
		if m := reNotice.FindStringSubmatch(text); m != nil {
			color.Printf("@{|w}%- 16s  %s\n", "NOTICE", text)
			continue
		}

		// Keys and values
		if m := reAltKeyValue.FindStringSubmatch(text); m != nil {
			k, v := addKey(m[1], res.Host), m[2]
			color.Printf("@{|w}%- 16s  @{c}%- 40s @{w}%s\n", "ALT_KEY_VALUE", k, v)
			continue
		}
		if m := reAltKey.FindStringSubmatch(text); m != nil {
			k := addKey(m[1], res.Host)
			color.Printf("@{|w}%- 16s  @{c}%s\n", "BARE_ALT_KEY", k)
			continue
		}
		if m := reKeyValue.FindStringSubmatch(text); m != nil {
			k, v := addKey(m[1], res.Host), m[2]
			color.Printf("@{|w}%- 16s  @{c}%- 40s @{w}%s\n", "KEY_VALUE", k, v)
			continue
		}
		if m := reBareKey.FindStringSubmatch(text); m != nil {
			k := addKey(m[1], res.Host)
			color.Printf("@{|w}%- 16s  @{c}%s\n", "BARE_KEY", k)
			continue
		}
		if m := reBareValue.FindStringSubmatch(text); m != nil {
			v := m[1]
			color.Printf("@{|w}%- 16s  @{c}%- 40s @{w}%s\n", "BARE_VALUE", "", v)
			continue
		}

		// Unknown
		color.Printf("@{|.}%- 16s  @{|.}%s\n", "UNKNOWN", text)
	}

	fmt.Printf("\n")
}

func addKey(k, host string) string {
	k = transformKey(k)
	if _, ok := keys[k]; !ok {
		keys[k] = host
	} else if !strings.Contains(keys[k], host) {
		keys[k] = keys[k] + "  " + host
	}
	return k
}

var (
	reStrip      = regexp.MustCompile(`[\.\(\)]`)
	reUnderscore = regexp.MustCompile(`\s+|/`)
)

func transformKey(k string) string {
	k = strings.TrimSpace(k)
	k = strings.ToUpper(k)
	k = reStrip.ReplaceAllLiteralString(k, "")
	k = reUnderscore.ReplaceAllLiteralString(k, "_")
	return k
}
