package stream

import (
	"net"
	"video-stream/log"
)

func getLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Error("Could not get IP", "msg", err.Error())
		return ""
	}

	if len(addrs) == 0 {
		log.Error("No network interfaces found")
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
    }

	// TODO: Return error properly
	log.Error("Could not find local IP address")
	return ""
}
