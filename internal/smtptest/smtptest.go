// Package smtptest runs a minimal in-process SMTP server for tests, so code that sends
// mail through the real dialer can be exercised end to end and its messages inspected.
// It speaks just enough SMTP for a plain, unauthenticated submission: no STARTTLS and
// no AUTH are advertised, so a dialer with an empty username sends in the clear.
package smtptest

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Message is one received mail.
type Message struct {
	From string
	To   []string
	Data string
}

// Server is a running fake SMTP server.
type Server struct {
	Host string
	Port int

	listener net.Listener
	mutex    sync.Mutex
	messages []Message
}

// Start listens on a random loopback port and stops the server on test cleanup.
func Start(t testing.TB) *Server {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("smtptest: failed to listen: %v", err)
	}

	address := listener.Addr().(*net.TCPAddr)
	server := &Server{Host: address.IP.String(), Port: address.Port, listener: listener}
	go server.serve()
	t.Cleanup(func() { _ = listener.Close() })
	return server
}

// Addr is host:port.
func (s *Server) Addr() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

// Messages returns a copy of every mail received so far.
func (s *Server) Messages() []Message {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]Message(nil), s.messages...)
}

func (s *Server) serve() {
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(connection)
	}
}

func (s *Server) handle(connection net.Conn) {
	defer connection.Close()

	reader := bufio.NewReader(connection)
	reply := func(line string) { _, _ = connection.Write([]byte(line + "\r\n")) }
	reply("220 smtptest ready")

	current := Message{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		command := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			reply("250 smtptest")
		case strings.HasPrefix(command, "MAIL FROM:"):
			current = Message{From: strings.Trim(line[len("MAIL FROM:"):], " <>")}
			reply("250 OK")
		case strings.HasPrefix(command, "RCPT TO:"):
			current.To = append(current.To, strings.Trim(line[len("RCPT TO:"):], " <>"))
			reply("250 OK")
		case command == "DATA":
			reply("354 end with <CRLF>.<CRLF>")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				data.WriteString(dataLine)
			}
			current.Data = data.String()
			s.mutex.Lock()
			s.messages = append(s.messages, current)
			s.mutex.Unlock()
			reply("250 OK queued")
		case command == "QUIT":
			reply("221 bye")
			return
		default:
			// RSET, NOOP and anything else: accept.
			reply("250 OK")
		}
	}
}
