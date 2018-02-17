// Package udp implements UDP test helpers. It lets you assert that certain
// strings must or must not be sent to a given local UDP listener.
package udp

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

var (
	addr     *string
	listener *net.UDPConn
	Timeout  time.Duration = time.Millisecond
	logBuf   []string
)

// TestingT is an interface wrapper around TestingT
// Makes this tester play nice with Ginkgo
type TestingT interface {
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
}

func resetLogBuf() {
	logBuf = []string{}
}

func errorF(format string, args ...interface{}) {
	logBuf = append(logBuf, fmt.Sprintf(format, args))
}

func emitLog(t TestingT) {
	if len(logBuf) > 0 {
		t.Error(strings.Join(logBuf, "\n"))
		resetLogBuf()
	}
}

type fn func()

// SetAddr sets the UDP port that will be listened on.
func SetAddr(a string) {
	addr = &a
}

func start(t TestingT) {
	resAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		t.Fatal(err)
	}
	listener, err = net.ListenUDP("udp", resAddr)
	if err != nil {
		t.Fatal(err)
	}
}

func stop(t TestingT) {
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func getMessage(t TestingT, body fn, expectData bool) string {
	start(t)
	defer stop(t)
	body()

	message := make([]byte, 1024*32)
	var bufLen int
	for {
		listener.SetReadDeadline(time.Now().Add(Timeout))
		n, _, err := listener.ReadFrom(message[bufLen:])
		if n == 0 {
			if err != nil && bufLen == 0 && expectData {
				errorF("Error reading udp data: %v", err)
			}
			break
		} else {
			bufLen += n
		}
	}
	msg := string(message[0:bufLen])
	return msg
}

func get(t TestingT, match string, body fn, expectData bool) (got string, equals bool, contains bool) {
	got = getMessage(t, body, expectData)
	equals = got == match
	contains = strings.Contains(got, match)
	return got, equals, contains
}

func printLocation(t TestingT) {
	_, file, line, _ := runtime.Caller(2)
	errorF("At: %s:%d", file, line)
}

// ShouldReceiveOnly will fire a test error if the given function doesn't send
// exactly the given string over UDP.
func ShouldReceiveOnly(t TestingT, expected string, body fn) {
	defer emitLog(t)
	got, equals, _ := get(t, expected, body, true)
	if !equals {
		printLocation(t)
		errorF("Expected: %#v", expected)
		errorF("But got: %#v", got)
	}
}

// ShouldNotReceiveOnly will fire a test error if the given function sends
// exactly the given string over UDP.
func ShouldNotReceiveOnly(t TestingT, notExpected string, body fn) {
	defer emitLog(t)
	_, equals, _ := get(t, notExpected, body, false)
	if equals {
		printLocation(t)
		errorF("Expected not to get: %#v", notExpected)
	}
}

// ShouldReceive will fire a test error if the given function doesn't send the
// given string over UDP.
func ShouldReceive(t TestingT, expected string, body fn) {
	defer emitLog(t)
	got, _, contains := get(t, expected, body, false)
	if !contains {
		printLocation(t)
		errorF("Expected: %#v", expected)
		errorF("But got: %#v", got)
	}
}

// ShouldNotReceive will fire a test error if the given function sends the
// given string over UDP.
func ShouldNotReceive(t TestingT, expected string, body fn) {
	defer emitLog(t)
	got, _, contains := get(t, expected, body, false)
	if contains {
		printLocation(t)
		errorF("Expected not to find: %#v", expected)
		errorF("But got: %#v", got)
	}
}

// ShouldReceiveNothing will fire a test error if the given function sends any
// data over UDP.
func ShouldReceiveNothing(t TestingT, body fn) {
	defer emitLog(t)
	got, _, _ := get(t, "", body, false)
	if len(got) > 0 {
		printLocation(t)
		errorF("Expected no data, but got: %#v", got)
	}
}

// ShouldReceiveAll will fire a test error unless all of the given strings are
// sent over UDP.
func ShouldReceiveAll(t TestingT, expected []string, body fn) {
	defer emitLog(t)
	got := getMessage(t, body, true)
	failed := false

	for _, str := range expected {
		if !strings.Contains(got, str) {
			if !failed {
				printLocation(t)
				failed = true
			}
			errorF("Expected to find: %#v", str)
		}
	}

	if failed {
		errorF("But got: %#v", got)
	}
}

// ShouldNotReceiveAny will fire a test error if any of the given strings are
// sent over UDP.
func ShouldNotReceiveAny(t TestingT, unexpected []string, body fn) {
	defer emitLog(t)
	got := getMessage(t, body, false)
	failed := false

	for _, str := range unexpected {
		if strings.Contains(got, str) {
			if !failed {
				printLocation(t)
				failed = true
			}
			errorF("Expected not to find: %#v", str)
		}
	}

	if failed {
		errorF("But got: %#v", got)
	}
}

func ShouldReceiveAllAndNotReceiveAny(t TestingT, expected []string, unexpected []string, body fn) {
	defer emitLog(t)
	got := getMessage(t, body, true)
	failed := false

	for _, str := range expected {
		if !strings.Contains(got, str) {
			if !failed {
				printLocation(t)
				failed = true
			}
			errorF("Expected to find: %#v", str)
		}
	}
	for _, str := range unexpected {
		if strings.Contains(got, str) {
			if !failed {
				printLocation(t)
				failed = true
			}
			errorF("Expected not to find: %#v", str)
		}
	}

	if failed {
		errorF("but got: %#v", got)
	}
}

func ReceiveString(t TestingT, body fn) string {
	return getMessage(t, body, true)
}
