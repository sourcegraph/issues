package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	requiredKeyLength = 32  // 32 bytes is the required length for AES-256.
	separator         = "$" // Used specifically because $ is not part of base64
)

// EncryptionError is an error about encryption or decryption.
type EncryptionError struct {
	error
}

// encryptor is an interface that provides encryption & decryption primitives.
type encryptor interface {
	// ConfiguredToEncrypt returns true if the encryptor is configured to encrypt.
	ConfiguredToEncrypt() bool
	// ConfiguredToRotate returns true if primary and secondary keys are both configured and valid keys.
	ConfiguredToRotate() bool
	// Decrypt decrypts a ciphertext and attempting all keys.
	Decrypt(ciphertext string) (string, error)
	// Encrypt encrypts a plaintext with the primary key.
	Encrypt(plaintext string) (string, error)
	// PrimaryKeyHash returns the hash of the primary key to be used for filtering.
	PrimaryKeyHash() string
	// SecondaryKeyHash returns the hash of secondary key to be used for filtering.
	SecondaryKeyHash() string
	// RotateEncryption decrypts given ciphertext and then re-encrypts with the primary key.
	RotateEncryption(ciphertext string) (string, error)
}

// aesGCMEncodedEncryptor is an encryptor that uses AES-GCM for encryption and decryption,
// and base64 to encode encrypted result.
type aesGCMEncodedEncryptor struct {
	// primaryKey is always used for encryption.
	primaryKey []byte
	// secondaryKey was the primary key that was used for encryption previously.
	secondaryKey []byte
	// primaryKeyHash contains a partial hash of the active encryption key (i.e. primary key).
	primaryKeyHash string
	// secondaryKeyHash contains a partial hash of the previously used encryption key (i.e. secondary key).
	secondaryKeyHash string
}

func newAESGCMEncodedEncryptor(primaryKey, secondaryKey []byte) encryptor {
	return aesGCMEncodedEncryptor{
		primaryKey:       primaryKey,
		secondaryKey:     secondaryKey,
		primaryKeyHash:   sliceKeyHash(primaryKey),
		secondaryKeyHash: sliceKeyHash(secondaryKey),
	}
}

// sliceKeyHash returns a string which is the first 6 bytes of given key's SHA2-256 checksum in hexadecimal.
// This checksum was chosen due to the (current) lack of SHA2 collisions, and is used to indicate without
// leaking, how something has encrypted. The inspiration for this is /etc/shadow.
func sliceKeyHash(k []byte) string {
	if k == nil {
		return ""
	}

	h := sha256.New()
	_, _ = h.Write(k)
	return hex.EncodeToString(h.Sum(nil))[:6]
}

// gcmEncrypt is the general purpose encryption function used to return the encrypted versions of bytes.
// gcmEncrypt uses 256-bit AES-GCM. This both hides the content of the data and provides a check that it
// hasn't been altered. Output takes the form `nonce|ciphertext|tag` where '|' indicates concatenation.
// It is a modified version of
// https://github.com/gtank/cryptopasta/blob/1f550f6f2f69009f6ae57347c188e0a67cd4e500/encrypt.go#L37
func gcmEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// gcmDecrypt decrypts data using 256-bit AES-GCM. This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input form
// `nonce|ciphertext|tag` where '|' indicates concatenation. It is a modified version of
// https://github.com/gtank/cryptopasta/blob/1f550f6f2f69009f6ae57347c188e0a67cd4e500/encrypt.go#L60
func gcmDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
}

// PrimaryKeyHash returns the keyHash for the primary key.
func (e aesGCMEncodedEncryptor) PrimaryKeyHash() string {
	return e.primaryKeyHash
}

// SecondaryKeyHash returns the keyHash for the secondary key.
func (e aesGCMEncodedEncryptor) SecondaryKeyHash() string {
	return e.secondaryKeyHash
}

// ConfiguredToEncrypt returns the status of our encryptor, whether or not
// it has a key specified, and can thus encrypt.
func (e aesGCMEncodedEncryptor) ConfiguredToEncrypt() bool {
	return len(e.primaryKey) == requiredKeyLength
}

// ConfiguredToRotate returns the status of our encryptor. If it contains two keys it
// is configured to rotate.
func (e aesGCMEncodedEncryptor) ConfiguredToRotate() bool {
	return len(e.primaryKey) == requiredKeyLength && len(e.secondaryKey) == requiredKeyLength
}

// Encrypt encrypts the plaintext using the primaryKey of the encryptor. This
// relies on the AES-GCM encryption defined in encrypt, within this package, and returns a base64 encoded string
func (e aesGCMEncodedEncryptor) Encrypt(plaintext string) (ciphertext string, err error) {
	if len(e.primaryKey) < requiredKeyLength {
		return "", &EncryptionError{errors.New("primary key is unavailable")}
	}

	cipherbytes, err := gcmEncrypt([]byte(plaintext), e.primaryKey)
	if err != nil {
		return "", &EncryptionError{errors.Errorf("unable to encrypt: %v", err)}
	}

	ciphertext = base64.StdEncoding.EncodeToString(cipherbytes)
	return e.PrimaryKeyHash() + separator + ciphertext, nil
}

var ErrDecryptAttemptedButFailed = &EncryptionError{
	errors.New("decrypt attempted but failed with both primary and secondary keys"),
}

// Decrypt decrypts base64 encoded ciphertext using the primaryKey of the encryptor
// This relies on AES-GCM. The `ErrDecryptAttemptedButFailed` is returned when attempts
// have been made to decrypt but failed with both primary and secondary keys.
func (e aesGCMEncodedEncryptor) Decrypt(ciphertext string) (plaintext string, err error) {
	if len(e.primaryKey) < requiredKeyLength && len(e.secondaryKey) < requiredKeyLength {
		return "", &EncryptionError{errors.New("no valid keys available")}
	}

	// If the ciphertext does not contain the separator, or has a new line,
	// it is definitely not encrypted.
	if !strings.Contains(ciphertext, separator) ||
		strings.Contains(ciphertext, "\n") {
		return ciphertext, nil
	}

	var keyToDecrypt []byte

	// Use the keyHash prefix to determine if we can decrypt it or whether it is encrypted at all.
	if strings.HasPrefix(ciphertext, e.PrimaryKeyHash()+separator) {
		ciphertext = ciphertext[len(e.PrimaryKeyHash()+separator):]
		keyToDecrypt = e.primaryKey
	} else if strings.HasPrefix(ciphertext, e.SecondaryKeyHash()+separator) {
		ciphertext = ciphertext[len(e.SecondaryKeyHash()+separator):]
		keyToDecrypt = e.secondaryKey
	} else {
		return "", ErrDecryptAttemptedButFailed
	}

	decodedCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", &EncryptionError{err}
	}

	plainbytes, err := gcmDecrypt(decodedCiphertext, keyToDecrypt)
	if err != nil {
		return "", &EncryptionError{err}
	}
	return string(plainbytes), nil
}

// RotateEncryption rotates the encryption on a ciphertext by decrypting the ciphertext
// using the primaryKey, and then reencrypting it using the secondaryKey.
func (e aesGCMEncodedEncryptor) RotateEncryption(ciphertext string) (string, error) {
	if !e.ConfiguredToRotate() {
		return "", &EncryptionError{errors.New("key rotation not configured")}
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return e.Encrypt(plaintext)
}

// noOpEncryptor always returns original content and does no encryption or decryption.
type noOpEncryptor struct{}

func (noOpEncryptor) PrimaryKeyHash() string {
	return ""
}

func (noOpEncryptor) SecondaryKeyHash() string {
	return ""
}

func (noOpEncryptor) Encrypt(s string) (string, error) {
	return s, nil
}

func (noOpEncryptor) Decrypt(s string) (string, error) {
	return s, nil
}

func (noOpEncryptor) ConfiguredToEncrypt() bool {
	return false
}

func (noOpEncryptor) ConfiguredToRotate() bool {
	return false
}

func (noOpEncryptor) RotateEncryption(s string) (string, error) {
	return s, nil
}

// ConfiguredToEncrypt returns a boolean indicating whether this type of
// encryption was configured to encrypt. This is effectively the status of
// a struct having an encryption key specified.
func ConfiguredToEncrypt() bool {
	return defaultEncryptor.ConfiguredToEncrypt()
}

// ConfiguredToRotate returns a boolean indicating whether this type of
// encryption was configured to rotate keys. This is effectively the status of
// a struct having two encryption keys specified.
func ConfiguredToRotate() bool {
	return defaultEncryptor.ConfiguredToRotate()
}

// Encrypt encrypts the plaintext and returns the encrypted value.
// An error is returned when encryption fails. It returns the original
// content if encryption is not configured.
func Encrypt(plaintext string) (string, error) {
	return defaultEncryptor.Encrypt(plaintext)
}

// Decrypt decrypts the ciphertext and returns the decrypted value.
// An error is returned when decryption fails. It returns the original
// content if encryption is not configured.
func Decrypt(ciphertext string) (string, error) {
	return defaultEncryptor.Decrypt(ciphertext)
}

// RotateEncryption rotates the encryption on a ciphertext by decrypting
// the ciphertext and then re-encrypting it using the primary key.
func RotateEncryption(ciphertext string) (string, error) {
	return defaultEncryptor.RotateEncryption(ciphertext)
}

// PrimaryKeyHash returns hash of the primary key to be used for filtering.
func PrimaryKeyHash() string {
	return defaultEncryptor.PrimaryKeyHash()
}

// SecondaryKeyHash returns hash of the secondary key to be used for filtering.
func SecondaryKeyHash() string {
	return defaultEncryptor.SecondaryKeyHash()
}
