package pilot_exchange_test

import (
	"testing"
	"github.com/jacksonzamorano/pilot/pilot-exchange"
)

type TestMessage struct {
	Message string `json:"message"`
}

func TestEndToEnd(t *testing.T) {
	value := TestMessage{Message: "Hello, world!"}
	encrypted := pilot_exchange.EncodeJson(value)
	decrypted := pilot_exchange.DecodeJson[TestMessage](encrypted)
	if value.Message != decrypted.Message {
		t.Error("Encryption/decryption failed.")
	}
}
