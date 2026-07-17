package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var store = map[string]string{}

func handleCommand(args []string) string {
	cmd := strings.ToUpper(args[0])

	switch cmd {
	case "PING":
		if len(args) > 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		if len(args) == 1 {
			return encodeSimpleString("PONG")
		}

		return encodeBulkString(args[1], true)
	case "ECHO":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		return encodeBulkString(args[1], true)
	case "COMMAND":
		return encodeSimpleString("OK")

	case "SET":
		if len(args) == 3 {
			store[args[1]] = args[2]
			return encodeSimpleString("OK")
		}
		if len(args) == 4 {
			_, exist := store[args[1]]
			if (strings.ToUpper(args[3]) == "NX" && exist) || (strings.ToUpper(args[3]) == "XX" && !exist) {
				return encodeBulkString("", false)
			}

			store[args[1]] = args[2]
			return encodeSimpleString("OK")
		}

		return errorWrongNumberOfArguments(cmd)
	case "GET":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		val, ok := store[args[1]]
		return encodeBulkString(val, ok)

	case "DBSIZE":
		if len(args) != 1 {
			return errorWrongNumberOfArguments(cmd)
		}

		return encodeNumber(len(store))
	case "INCR", "DECR":
		valStr, exist := store[args[1]]
		if !exist {
			valStr = "0"
		}

		val, err := strconv.Atoi(valStr)
		if err != nil {
			return encodeError("value is not an integer or out of range")
		}

		var newVal int
		if cmd == "INCR" {
			newVal = val + 1
		} else {
			newVal = val - 1
		}

		store[args[1]] = strconv.Itoa(newVal)
		return encodeNumber(newVal)
	case "INCRBY", "DECRBY":
		valStr, exist := store[args[1]]
		if !exist {
			valStr = "0"
		}

		val, err := strconv.Atoi(valStr)
		if err != nil {
			return encodeError("value is not an integer or out of range")
		}

		amount, err := strconv.Atoi(args[2])
		if err != nil {
			return encodeError("amount is not an integer")
		}

		var newVal int
		if cmd == "INCRBY" {
			newVal = val + amount
		} else {
			newVal = val - amount
		}

		store[args[1]] = strconv.Itoa(newVal)
		return encodeNumber(newVal)
	}

	return fmt.Sprintf("-ERR unknown command '%s'\r\n", cmd)
}

func encodeNumber(n int) string {
	return fmt.Sprintf(":%d\r\n", n)
}

func encodeSimpleString(s string) string {
	return fmt.Sprintf("+%s\r\n", s)
}

func encodeBulkString(s string, ok bool) string {
	if !ok {
		return "$-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func encodeError(s string) string {
	return fmt.Sprintf("-ERR %s\r\n", s)
}

func errorWrongNumberOfArguments(command string) string {
	return encodeError(fmt.Sprintf("wrong number of arguments for '%s' command", command))
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		args := parseArgs(line)
		response := handleCommand(args)
		fmt.Print(response)
	}
}

func parseArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	for _, ch := range line {
		switch {
		case ch == '"' && !inQuotes:
			inQuotes = true
		case ch == '"':
			inQuotes = false
		case ch == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
