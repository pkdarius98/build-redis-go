package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var store = map[string]string{}

var expiryTimes = map[string]time.Time{}

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
		if len(args) == 4 || len(args) == 6 {
			_, exist := store[args[1]]
			condition := strings.ToUpper(args[len(args)-1])
			if (condition == "NX" && exist) || (condition == "XX" && !exist) {
				return encodeBulkString("", false)
			}
		}

		if len(args) == 3 || len(args) == 4 {
			store[args[1]] = args[2]
			return encodeSimpleString("OK")
		}

		if len(args) == 5 || len(args) == 6 {
			expiryDuration, err := strconv.Atoi(args[4])
			if err != nil {
				return encodeError("value is not an integer or out of range")
			}

			now := time.Now()

			var expireDurationUnit time.Duration
			if args[3] == "EX" {
				expireDurationUnit = time.Second / time.Nanosecond
			} else {
				expireDurationUnit = time.Millisecond / time.Nanosecond
			}

			expiryTime := now.Add(time.Duration(expiryDuration) * expireDurationUnit)
			expiryTimes[args[1]] = expiryTime

			store[args[1]] = args[2]
			return encodeSimpleString("OK")
		}

		return errorWrongNumberOfArguments(cmd)

	case "GET":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		key := args[1]
		val, ok := store[key]

		if !ok {
			return encodeBulkString(val, ok)
		}

		expiryTime, ok := expiryTimes[key]
		if !ok {
			return encodeBulkString(val, true)
		}

		now := time.Now()
		if expiryTime.After(now) {
			return encodeBulkString(val, true)
		}

		delete(expiryTimes, key)
		delete(store, key)
		return encodeBulkString("", false)
	case "DBSIZE":
		if len(args) != 1 {
			return errorWrongNumberOfArguments(cmd)
		}

		var count int
		now := time.Now()
		for key, _ := range store {
			expiryTime, exist := expiryTimes[key]
			if !exist {
				count++
				continue
			}

			if expiryTime.Before(now) {
				continue
			}
			count++
		}

		return encodeNumber(count)

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

	case "EXPIRE":
		if len(args) != 3 {
			return errorWrongNumberOfArguments(cmd)
		}

		_, exist := store[args[1]]
		if !exist {
			return encodeNumber(0)
		}

		seconds, err := strconv.Atoi(args[2])
		if err != nil {
			return encodeError("value is not an integer or out of range")
		}

		now := time.Now()
		expiryTime := now.Add(time.Duration(seconds) * (time.Second / time.Nanosecond))
		expiryTimes[args[1]] = expiryTime
		return encodeNumber(1)

	case "TTL", "PTTL":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		_, exist := store[args[1]]
		if !exist {
			return encodeNumber(-2)
		}

		expiryTime, exist := expiryTimes[args[1]]
		if !exist {
			return encodeNumber(-1)
		}

		now := time.Now()
		if !expiryTime.After(now) {
			return encodeNumber(-2)
		}

		if cmd == "TTL" {
			return encodeNumber(int(expiryTime.Sub(time.Now()).Round(time.Second).Seconds()))
		}

		return encodeNumber(int(expiryTime.Sub(time.Now()).Round(time.Millisecond).Milliseconds()))

	case "PERSIST":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		_, exist := store[args[1]]
		if !exist {
			return encodeNumber(0)
		}

		delete(expiryTimes, args[1])
		return encodeNumber(1)

	case "WAIT":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		seconds, err := strconv.Atoi(args[1])
		if err != nil {
			return encodeError("value is not an integer or out of range")
		}

		time.Sleep(time.Duration(seconds) * time.Millisecond)
		return encodeSimpleString("OK")
	case "EXISTS":
		if len(args) != 2 {
			return errorWrongNumberOfArguments(cmd)
		}

		key := args[1]
		_, ok := store[key]

		if !ok {
			return encodeNumber(0)
		}

		expiryTime, ok := expiryTimes[key]
		if !ok {
			return encodeNumber(1)
		}

		now := time.Now()
		if expiryTime.After(now) {
			return encodeNumber(1)
		}

		delete(store, key)
		delete(expiryTimes, key)

		return encodeNumber(0)
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
