package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var store = map[string]string{}

func encodeBulk(s string, ok bool) string {
	if !ok {
		return "$-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}
func encodeSimple(s string) string { return "+" + s + "\r\n" }
func encodeError(s string) string  { return "-" + s + "\r\n" }

func handle(args []string) string {
	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "PING":
		if len(args) == 1 {
			return encodeSimple("PONG")
		}
		return encodeBulk(args[1], true)
	case "ECHO":
		return encodeBulk(args[1], true)
	case "SET":
		store[args[1]] = args[2]
		return encodeSimple("OK")
	case "GET":
		v, ok := store[args[1]]
		return encodeBulk(v, ok)
	}
	return encodeError(fmt.Sprintf("ERR unknown command '%s'", cmd))
}

// parseRequest reads one RESP array from r and returns its arg list.
func parseRequest(r *bufio.Reader) ([]string, error) {
	// Read the *N\r\n line
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected '*', got %q", line)
	}

	n, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
	if err != nil {
		return nil, err
	}

	var args []string
	for i := 0; i < n; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(line, "$") {
			return nil, fmt.Errorf("expected '$', got %q", line)
		}

		lenBytes, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
		if err != nil {
			return nil, err
		}

		buf := make([]byte, lenBytes+2)
		bytesRead, err := io.ReadFull(r, buf)
		if err != nil || bytesRead != lenBytes+2 {
			return nil, err
		}

		args = append(args, string(buf[:lenBytes]))

	}

	return args, nil
}

func main() {
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for {
		args, err := parseRequest(r)
		if err != nil {
			return
		}
		w.WriteString(handle(args))
		w.Flush()
	}
}
