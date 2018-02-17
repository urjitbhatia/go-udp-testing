package udp

import (
	"net"
	"testing"
	"time"
)

var (
	testAddr = ":8126"
)

func setup(t *testing.T) net.Conn {
	udpClient, err := net.DialTimeout("udp", testAddr, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	SetAddr(testAddr)

	return udpClient
}

func TestAll(t *testing.T) {
	udpClient := setup(t)

	testValues := [][]interface{}{
		[]interface{}{"foo", "foo", true, true},
		[]interface{}{"foo", "bar", false, false},
		[]interface{}{"foo", "foobar", false, true},
		[]interface{}{"foo", "", false, false},
		[]interface{}{"", "", true, true},
	}

	for _, values := range testValues {
		shouldGet := values[0].(string)
		sendString := values[1].(string)
		shouldEquals := values[2].(bool)
		shouldContains := values[3].(bool)

		got, equals, contains := get(t, shouldGet, func() {
			udpClient.Write([]byte(sendString))
		}, true)

		if got != sendString {
			t.Errorf("Should've got %#v but got %#v", sendString, got)
		}
		if equals != shouldEquals {
			t.Errorf("Equals should've been %#v but was %#v", shouldEquals, equals)
		}
		if contains != shouldContains {
			t.Errorf("Contains should've been %#v but was %#v", shouldContains, contains)
		}
	}

	ShouldReceiveOnly(t, "foo", func() {
		udpClient.Write([]byte("foo"))
	})

	ShouldNotReceiveOnly(t, "bar", func() {
		udpClient.Write([]byte("foo"))
	})

	ShouldReceive(t, "foo", func() {
		udpClient.Write([]byte("barfoo"))
	})

	ShouldNotReceive(t, "bar", func() {
		udpClient.Write([]byte("fooba"))
	})

	ShouldReceiveAll(t, []string{"foo", "bar"}, func() {
		udpClient.Write([]byte("foobizbar"))
	})

	ShouldNotReceiveAny(t, []string{"fooby", "bars"}, func() {
		udpClient.Write([]byte("foobizbar"))
	})

	ShouldReceiveAllAndNotReceiveAny(t, []string{"foo", "bar"}, []string{"fooby", "bars"}, func() {
		udpClient.Write([]byte("foo"))
		udpClient.Write([]byte("biz"))
		udpClient.Write([]byte("bar"))
	})

	ShouldReceiveNothing(t, func() {})

	// This should fail, but it also shouldn't stall out
	// ShouldReceive(t, "foo", func() {})
}

func TestRaceConditionInReadingResults(t *testing.T) {
	udpClient := setup(t)

	ShouldReceiveAllAndNotReceiveAny(t, []string{"foo", "bar", "biz"}, []string{"fooby", "bars"}, func() {
		time.Sleep(time.Millisecond * 100)
		udpClient.Write([]byte("foo"))
		time.Sleep(time.Millisecond * 200)
		udpClient.Write([]byte("biz"))
		time.Sleep(time.Millisecond * 500)
		udpClient.Write([]byte("bar"))
	})
}
