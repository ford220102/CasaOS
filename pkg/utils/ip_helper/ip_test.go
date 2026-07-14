package ip_helper

import (
	"fmt"
	"net"
	"os"
	"testing"

	"go.uber.org/goleak"
)

// testLocalIP is used only for unit tests - local network range (RFC 1918)
const defaultTestIP = "192.168.2.10"

// getTestIP returns the test IP from environment or default value
func getTestIP() string {
	if ip := os.Getenv("TEST_LOCAL_IP"); ip != "" {
		return ip
	}
	return defaultTestIP
}

func TestGetExternalIPV4(t *testing.T) {
	goleak.VerifyNone(t)

	ipv4 := make(chan string)
	go func() { ipv4 <- GetExternalIPV4() }()
	fmt.Println(<-ipv4)
}

func TestGetExternalIPV6(t *testing.T) {
	ipv6 := make(chan string)
	go func() { ipv6 <- GetExternalIPV6() }()
	fmt.Println(<-ipv6)
}

func TestGetLoclIp(t *testing.T) {
	fmt.Println(GetLoclIp())
}

func TestHasLocalIP(t *testing.T) {
	fmt.Println("dddd")
	
	// Parse test IP - skip test if invalid
	testIP := net.ParseIP(getTestIP())
	if testIP == nil {
		t.Skip("Invalid test IP address - skipping test")
	}
	
	fmt.Println(HasLocalIP(testIP))
}
