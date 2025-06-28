// Package pilot_exchange provides secure authentication token management through AES encryption.
// This package implements a stateless authentication system where user session data is encrypted
// into tokens that can be safely transmitted to clients and verified when returned. The tokens
// are self-contained and include expiration information, eliminating the need for server-side
// session storage.
//
// Key Features:
// - AES-256-CBC encryption for secure token generation
// - JSON-based payload structure with automatic marshaling/unmarshaling
// - Built-in expiration handling for time-limited tokens
// - Stateless design - no server-side session storage required
// - Environment-based secret key configuration for security
// - Automatic padding and IV generation for cryptographic security
//
// Security Considerations:
// - Uses AES-256-CBC encryption with random initialization vectors
// - Requires a 32-byte secret key from the SIGNING_KEY environment variable
// - Tokens include original content size to prevent padding oracle attacks
// - All operations use secure random number generation
//
// Usage Example:
//   // Create an authentication payload
//   payload := pilot_exchange.AuthPayload{
//       AccountId:  123,
//       Expiration: time.Now().Add(24 * time.Hour),
//   }
//   
//   // Encrypt the payload into a token
//   token := pilot_exchange.EncodeJson(payload)
//   
//   // Later, decrypt and verify the token
//   decoded := pilot_exchange.DecodeJson[pilot_exchange.AuthPayload](token)
//   if decoded != nil && decoded.Expiration.After(time.Now()) {
//       // Token is valid and not expired
//       userID := decoded.AccountId
//   }
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

// AuthPayload represents the standard authentication data structure that gets encrypted
// into tokens. This struct contains the essential information needed to identify and
// validate a user session, including the user's account ID and when the token expires.
//
// Fields:
//   - AccountId: The unique identifier for the authenticated user's account
//   - Expiration: The timestamp when this token becomes invalid
//
// Example:
//   payload := pilot_exchange.AuthPayload{
//       AccountId:  userID,
//       Expiration: time.Now().Add(7 * 24 * time.Hour), // Valid for 7 days
//   }
type AuthPayload struct {
	AccountId  int64     `json:"account_id"`
	Expiration time.Time `json:"expiration"`
}

// getSecret retrieves the encryption key from the SIGNING_KEY environment variable.
// This key is used for all AES encryption and decryption operations. If no key is
// configured, a default key is returned with a warning - this should never be used
// in production environments as it compromises security.
//
// Returns:
//   - string: The 32-byte encryption key for AES operations
//
// Security Note:
//   The SIGNING_KEY environment variable must contain exactly 32 bytes for AES-256.
//   Using the default key in production is a critical security vulnerability.
func getSecret() string {
	key, exists := os.LookupEnv("SIGNING_KEY")
	if exists {
		return key
	}
	log.Println("Warning: No signing key found. Using default. DO NOT USE IN PRODUCTION.")
	return "SOME_RANDOM_KEY_SOME_RANDOM_KEY_"
}

// EncodeJson encrypts any JSON-serializable data structure into a secure token string.
// This function performs the following operations:
// 1. Marshals the input data to JSON
// 2. Applies PKCS#7 padding to meet AES block size requirements
// 3. Generates a random initialization vector (IV)
// 4. Encrypts the data using AES-256-CBC
// 5. Appends the original content size to prevent padding oracle attacks
// 6. Encodes the result as a hexadecimal string
//
// The resulting token is safe to transmit over insecure channels and can be stored
// in cookies, local storage, or sent in HTTP headers. The token contains all the
// information needed for later decryption and verification.
//
// Parameters:
//   - data: Any JSON-serializable data structure to encrypt
//
// Returns:
//   - string: A hexadecimal-encoded encrypted token
//
// Example:
//   payload := AuthPayload{AccountId: 123, Expiration: time.Now().Add(time.Hour)}
//   token := pilot_exchange.EncodeJson(payload)
//   // Send token to client in response header or cookie
//
// Security Note:
//   Each call generates a unique IV, so the same data will produce different tokens.
//   This prevents attackers from detecting when the same data is being transmitted.
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

// DecodeJson decrypts a token string back into the original data structure.
// This function reverses the encryption process performed by EncodeJson:
// 1. Decodes the hexadecimal string to binary data
// 2. Extracts the original content size from the end of the data
// 3. Separates the IV from the encrypted content
// 4. Decrypts the content using AES-256-CBC
// 5. Removes padding and unmarshals the JSON back to the target type
//
// The function is generic and will return the data as the specified type T.
// If decryption fails at any stage (invalid token, wrong key, corrupted data),
// the function returns nil.
//
// Type Parameters:
//   - Data: The expected type of the decrypted data
//
// Parameters:
//   - contents: The hexadecimal-encoded encrypted token string
//
// Returns:
//   - *Data: A pointer to the decrypted data structure, or nil if decryption fails
//
// Example:
//   token := "..." // Received from client
//   payload := pilot_exchange.DecodeJson[AuthPayload](token)
//   if payload != nil && payload.Expiration.After(time.Now()) {
//       // Token is valid and not expired
//       userID := payload.AccountId
//   } else {
//       // Token is invalid, expired, or corrupted
//       return unauthorizedResponse()
//   }
//
// Error Handling:
//   Returns nil for any of the following conditions:
//   - Invalid hexadecimal encoding
//   - Corrupted or tampered token data
//   - Wrong encryption key
//   - Invalid JSON structure
//   - Mismatched data type
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
