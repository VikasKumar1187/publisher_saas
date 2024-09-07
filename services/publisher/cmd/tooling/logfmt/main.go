package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var service string

func init() {
	flag.StringVar(&service, "service", "", "filter which service to see")
}

func main() {
	flag.Parse()
	var b strings.Builder

	service := strings.ToLower(service)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		s := scanner.Text()

		m := make(map[string]interface{})
		err := json.Unmarshal([]byte(s), &m)
		if err != nil {
			if service == "" {
				fmt.Println(s)
			}
			continue
		}

		// If a service filter was provided, check.
		if service != "" && strings.ToLower(m["service"].(string)) != service {
			continue
		}

		// I like always having a traceid present in the logs.
		traceID := "00000000000000000000000000000000"
		if v, ok := m["trace_id"]; ok {
			traceID = fmt.Sprintf("%v", v)
		}

		// Build out the known portions of the log in the order I want them in.
		b.Reset()
		b.WriteString(fmt.Sprintf("%s: %s: %s: %s: %s: %s: ",
			m["service"],
			m["ts"],
			m["level"],
			traceID,
			m["caller"],
			m["msg"],
		))

		// Add specific fields in the order you want them to appear.
		if method, ok := m["method"]; ok {
			b.WriteString(fmt.Sprintf("method[%v]: ", method))
		}
		if path, ok := m["path"]; ok {
			b.WriteString(fmt.Sprintf("path[%v]: ", path))
		}
		if remoteAddr, ok := m["remoteaddr"]; ok {
			b.WriteString(fmt.Sprintf("remoteaddr[%v]: ", remoteAddr))
		}

		// Add the rest of the keys, ignoring the ones already added.
		for k, v := range m {
			switch k {
			case "service", "ts", "level", "trace_id", "caller", "msg", "method", "path", "remoteaddr":
				continue
			}
			// It's nice to see the key[value] in this format.
			b.WriteString(fmt.Sprintf("%s[%v]: ", k, v))
		}

		// Write the new log format, removing the last colon.
		out := b.String()
		fmt.Println(out[:len(out)-2])
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
