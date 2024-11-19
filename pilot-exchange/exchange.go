package pilot_exchange

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
)

type AuthPayload struct {
	AccountId  int64     `json:"account_id"`
	Expiration time.Time `json:"expiration"`
}

func getSecret() string {
	key, exists := os.LookupEnv("SIGNING_KEY")
	if exists {
		return key
	}
	log.Println("Warning: No signing key found. Using default. DO NOT USE IN PRODUCTION.")
	return "SOME_RANDOM_KEY_SOME_RANDOM_KEY_"
}

func EncodeJson(data interface{}) string {
	contents, _ := json.Marshal(data)
	content_size := len(contents)
	block, err := aes.NewCipher([]byte(getSecret()))
	if err != nil {
		log.Println("Couldn't create cipher for encryption.")
	}
	for len(contents)%aes.BlockSize != 0 {
		contents = append(contents, 0)
	}
	encrypted := make([]byte, aes.BlockSize+len(contents))
	if _, err := io.ReadFull(rand.Reader, encrypted[:aes.BlockSize]); err != nil {
		log.Println("Couldn't create iv for encryption.")
	}
	mode := cipher.NewCBCEncrypter(block, encrypted[:aes.BlockSize])
	mode.CryptBlocks(encrypted[aes.BlockSize:], contents)
	encrypted = append(encrypted, byte(content_size))
	return hex.EncodeToString(encrypted)
}

func DecodeJson[Data any](contents string) *Data {
	encrypted, err := hex.DecodeString(contents)
	if err != nil {
		log.Println("Couldn't decode contents of encrypted blob.")
		return nil
	}
	message_end := len(encrypted) - 1
	content_size := int(encrypted[message_end])
	block, err := aes.NewCipher([]byte(getSecret()))
	if err != nil {
		log.Println("Couldn't create cipher for decryption.")
		return nil
	}
	var decrypted []byte = make([]byte, len(encrypted)-aes.BlockSize-1)
	decrypter := cipher.NewCBCDecrypter(block, encrypted[:aes.BlockSize])
	decrypter.CryptBlocks(decrypted, encrypted[aes.BlockSize:message_end])
	var jsonDec Data
	jsonErr := json.Unmarshal(decrypted[:content_size], &jsonDec)
	if jsonErr != nil {
		log.Println("Couldn't unmarshal json.")
		return nil
	}
	return &jsonDec
}
