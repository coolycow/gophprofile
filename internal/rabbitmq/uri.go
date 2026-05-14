package rabbitmq

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// vhostToURIFragment кодирует виртуальный хост для AMQP URI (один сегмент пути, без двойного encoding).
func vhostToURIFragment(vhost string) string {
	s := strings.TrimSpace(vhost)
	if s == "" || s == "/" {
		return "%2F"
	}
	s = strings.TrimPrefix(s, "/")
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "/", "%2F")
	return s
}

// BuildURI собирает AMQP URI с корректным экранированием user/password/vhost.
func BuildURI(user, password, host string, port int, vhost string) string {
	hostport := net.JoinHostPort(host, strconv.Itoa(port))

	auth := ""
	if user != "" || password != "" {
		auth = url.UserPassword(user, password).String() + "@"
	}

	return fmt.Sprintf("amqp://%s%s/%s", auth, hostport, vhostToURIFragment(vhost))
}
