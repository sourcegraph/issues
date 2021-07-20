package mounted

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"hash/crc32"
	"io"
	"os"
	"strings"

	"github.com/sourcegraph/sourcegraph/schema"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/encryption"
)

func NewKey(ctx context.Context, k schema.MountedEncryptionKey) (*Key, error) {
	var secret []byte
	if k.EnvVarName != "" && k.Filepath == "" {
		secret = []byte(os.Getenv(k.EnvVarName))

	} else if k.Filepath != "" && k.EnvVarName == "" {
		keyBytes, err := os.ReadFile(k.Filepath)
		if err != nil {
			return nil, errors.Errorf("error reading secret file for %q: %v", k.Keyname, err)
		}
		secret = keyBytes
	} else {
		// Either the user has set none of EnvVarName or Filepath or both in their config. Either way we return an error.
		return nil, errors.Errorf(
			"must use only one of EnvVarName and Filepath, EnvVarName: %q, Filepath: %q",
			k.EnvVarName, k.Filepath,
		)
	}

	if len(secret) != 32 {
		return nil, errors.Errorf("invalid key length: %d, expected 32 bytes", len(secret))
	}

	return &Key{
		keyname: k.Keyname,
		version: k.Version,
		secret:  secret,
	}, nil
}

// Key is an encryption.Key implementation that uses AES GCM encryption, using a
// secret loaded either from an env var or a file
type Key struct {
	keyname string
	secret  []byte
	version string
}

func (k *Key) Version(ctx context.Context) (encryption.KeyVersion, error) {
	return encryption.KeyVersion{
		Type:    "mounted",
		Name:    k.keyname,
		Version: k.version,
	}, nil
}

func (k *Key) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.secret)
	if err != nil {
		return nil, errors.Wrap(err, "creating AES cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "creating GCM block cipher")
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	out := encryptedValue{
		KeyName:    k.keyname,
		Ciphertext: ciphertext,
		Checksum:   crc32Sum(plaintext),
	}
	jsonKey, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	buf := base64.StdEncoding.EncodeToString(jsonKey)
	return []byte(buf), err
}

func (k *Key) Decrypt(ctx context.Context, ciphertext []byte) (*encryption.Secret, error) {
	block, err := aes.NewCipher(k.secret)
	if err != nil {
		return nil, errors.Wrap(err, "creating AES cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "creating GCM block cipher")
	}

	buf, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, err
	}
	// unmarshal the encrypted value into encryptedValue, this struct contains the raw
	// ciphertext, the key name, and a crc32 checksum
	ev := encryptedValue{}
	err = json.Unmarshal(buf, &ev)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(ev.KeyName, k.keyname) {
		return nil, errors.New("invalid key name, are you trying to decrypt something with the wrong key?")
	}

	if len(ev.Ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}
	plaintext, err := gcm.Open(nil, ev.Ciphertext[:gcm.NonceSize()], ev.Ciphertext[gcm.NonceSize():], nil)
	if err != nil {
		return nil, err
	}

	if crc32Sum(plaintext) != ev.Checksum {
		return nil, errors.New("invalid checksum, either the wrong key was used, or the request was corrupted in transit")
	}
	s := encryption.NewSecret(string(plaintext))
	return &s, nil
}

type encryptedValue struct {
	KeyName    string
	Ciphertext []byte
	Checksum   uint32
}

func crc32Sum(data []byte) uint32 {
	t := crc32.MakeTable(crc32.Castagnoli)
	return crc32.Checksum(data, t)
}
