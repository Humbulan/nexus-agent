package main

import (
"encoding/json"
"flag"
"fmt"
"os"
"strconv"
"strings"
)

var checkOnly bool

func init() {
flag.BoolVar(&checkOnly, "check", false, "validate config and exit")
}

func runCheckMode(path string) int {
data, err := os.ReadFile(path)
if err != nil {
fmt.Fprintln(os.Stderr, "FAIL: cannot read", path, ":", err)
return 1
}
var raw []map[string]any
if err := json.Unmarshal(data, &raw); err != nil {
fmt.Fprintln(os.Stderr, "FAIL: invalid JSON:", err)
return 1
}
seen := map[string]bool{}
var problems []string
for i, s := range raw {
name, _ := s["name"].(string)
port, _ := s["port"].(string)
proto, _ := s["protocol"].(string)
if name == "" {
problems = append(problems, fmt.Sprintf("entry %d: missing name", i))
continue
}
if seen[name] {
problems = append(problems, fmt.Sprintf("duplicate name: %s", name))
}
seen[name] = true
p, err := strconv.Atoi(port)
if err != nil || p < 1 || p > 65535 {
problems = append(problems, fmt.Sprintf("%s: invalid port %q", name, port))
}
pl := strings.ToLower(proto)
if pl != "tcp" && pl != "udp" {
problems = append(problems, fmt.Sprintf("%s: protocol must be tcp or udp, got %q", name, proto))
}
}
if len(problems) > 0 {
fmt.Fprintln(os.Stderr, "FAIL:")
for _, p := range problems {
fmt.Fprintln(os.Stderr, "  -", p)
}
return 1
}
fmt.Printf("OK: %d services, 0 duplicate names, 0 invalid ports\n", len(raw))
return 0
}
