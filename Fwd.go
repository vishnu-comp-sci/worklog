package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"golang.org/x/net/ipv4"
)

const ssdpAddress = "239.255.255.250:1900"

func resolveUDPAddr() (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp4", ssdpAddress)
}

func listenUDP() (*net.UDPConn, error) {
	return net.ListenUDP("udp4", nil)
}

func createPacketConn(conn *net.UDPConn) (*ipv4.PacketConn, error) {
	return ipv4.NewPacketConn(conn), nil
}

func joinMulticastGroup(pConn *ipv4.PacketConn, udpAddr *net.UDPAddr) error {
	return pConn.JoinGroup(nil, udpAddr)
}

func createSSDPRequest() (*http.Request, error) {
	request, err := http.NewRequest("M-SEARCH", "*", nil)
	if err != nil {
		return nil, err
	}
	request.Host = ssdpAddress
	request.Header["ST"] = []string{"ssdp:all"}
	request.Header["MAN"] = []string{"\"ssdp:discover\""}
	request.Header["MX"] = []string{"5"}
	return request, nil
}

func sendSSDPRequest(pConn *ipv4.PacketConn, udpAddr *net.UDPAddr, request *http.Request) error {
	raw, err := httputil.DumpRequest(request, true)
	if err != nil {
		return err
	}
	_, err = pConn.WriteTo(raw, nil, udpAddr)
	return err
}

func readSSDPResponse(pConn *ipv4.PacketConn) ([]byte, *net.UDPAddr, error) {
	buffer := make([]byte, 1024)
	n, _, addr, err := pConn.ReadFrom(buffer)
	if err != nil {
		return nil, nil, err
	}
	return buffer[:n], addr.(*net.UDPAddr), nil
}

// ParseSSDPResponse parses SSDP response and extracts location URL.
func ParseSSDPResponse(response []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(response))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "LOCATION: ") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				return parts[1], nil
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("location not found in SSDP response")
}

// AddPortMapping sends a SOAP request to add a port mapping.

func addPortMapping(controlURL string, internalIP string, externalPort int, internalPort int) error {

	headers := map[string]string{
		"Content-Type": "text/xml; charset=\"utf-8\"",
		"Connection":   "close",
		"SOAPAction":   "\"urn:schemas-upnp-org:service:WANIPConnection:1#AddPortMapping\"",
	}

	soapBody := `<?xml version="1.0"?>
  <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
  <u:AddPortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1">
	<NewRemoteHost></NewRemoteHost>
	<NewExternalPort>%d</NewExternalPort>
	<NewProtocol>TCP</NewProtocol>
	<NewInternalPort>%d</NewInternalPort>
	<NewInternalClient>%s</NewInternalClient>
	<NewEnabled>1</NewEnabled>
	<NewPortMappingDescription>upnp</NewPortMappingDescription>
	<NewLeaseDuration>0</NewLeaseDuration>
  </u:AddPortMapping>
  </s:Body>
  </s:Envelope>`

	requestBody := fmt.Sprintf(soapBody, externalPort, internalPort, internalIP)

	req, err := http.NewRequest("POST", controlURL, strings.NewReader(requestBody))
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Status)
	fmt.Println(string(bodyBytes))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Port mapping added successfully")
		return nil
	} else {
		fmt.Println("Failed to add port mapping")
		return fmt.Errorf("Error adding port mapping: %s", resp.Status)
	}
}

func main() {
	udpAddr, err := resolveUDPAddr()
	if err != nil {
		panic(err)
	}

	conn, err := listenUDP()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	pConn, err := createPacketConn(conn)
	if err != nil {
		panic(err)
	}

	err = joinMulticastGroup(pConn, udpAddr)
	if err != nil {
		panic(err)
	}

	request, err := createSSDPRequest()
	if err != nil {
		panic(err)
	}

	err = sendSSDPRequest(pConn, udpAddr, request)
	if err != nil {
		panic(err)
	}

	response, _, err := readSSDPResponse(pConn)
	if err != nil {
		panic(err)
	}

	controlURL, err := ParseSSDPResponse(response)
	if err != nil {
		panic(err)
	}
	fmt.Println("Parsed control URL:", controlURL)

	// These values need to be set according to your requirements.
	internalClientIP := "192.168.1.2" // The IP of the device you want to forward the port to.
	externalPort := 27015             // The port you want to expose.
	internalPort := 27015             // The port on the internal device.

	err = addPortMapping(controlURL, internalClientIP, externalPort, internalPort)
	if err != nil {
		panic(err)
	}
}
