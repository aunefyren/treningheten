package smtptest

import (
	"net/smtp"
	"strings"
	"testing"
)

func TestServerRecordsAPlainSubmission(t *testing.T) {
	server := Start(t)

	err := smtp.SendMail(server.Addr(), nil, "from@test.local", []string{"a@test.local", "b@test.local"},
		[]byte("Subject: hello\r\n\r\nbody line\r\n"))
	if err != nil {
		t.Fatalf("SendMail: %v", err)
	}

	messages := server.Messages()
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	message := messages[0]
	if message.From != "from@test.local" || len(message.To) != 2 || !strings.Contains(message.Data, "body line") {
		t.Errorf("unexpected message: %+v", message)
	}
}
