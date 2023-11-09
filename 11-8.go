package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	// Specify the service to search for, replace with desired service.
	service := "ssdp:all"

	// Create a UDP address for SSDP multicast.
	ssdp, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		panic(err)
	}

	// Open a UDP connection.
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Set a deadline for the read operation.
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))

	// Send an M-SEARCH request.
	msg := []byte(
		"M-SEARCH * HTTP/1.1\r\n" +
			"HOST: 239.255.255.250:1900\r\n" +
			"ST:" + service + "\r\n" +
			"MAN: \"ssdp:discover\"\r\n" +
			"MX: 1\r\n" +
			"\r\n",
	)

	_, err = conn.WriteToUDP(msg, ssdp)
	if err != nil {
		panic(err)
	}

	// Listen for responses until the deadline is reached.
	for {
		buf := make([]byte, 2048)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break // Deadline reached, stop listening
			}
			panic(err)
		}

		fmt.Printf("Received response from %s:\n%s\n", addr, buf[:n])
	}
}
