package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	MS := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST:192.168.15.1:1900\r\n" +
		"ST:upnp:rootdevice\r\n" +
		"MX:2\r\n" +
		"MAN:\"ssdp:discover\"\r\n" +
		"\r\n"

	SOC, err := net.ListenPacket("udp", ":0")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer SOC.Close()

	SOC.SetDeadline(time.Now().Add(2 * time.Second))

	remoteAddr, err := net.ResolveUDPAddr("udp", "192.168.15.1:1900")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = SOC.WriteTo([]byte(MS), remoteAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		buffer := make([]byte, 8192)
		_, addr, err := SOC.ReadFrom(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			fmt.Println(err)
			return
		}
		fmt.Println(addr, string(buffer))
	}
}
