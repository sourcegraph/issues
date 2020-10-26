package secret

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateRandomAESKey(t *testing.T) {
	key, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != requiredKeyLength {
		t.Fatalf("Exepected key length of %d received %d", requiredKeyLength, len(key))
	}
}

var messageToEncrypt = "I Madam. I made radio, So I dared. Am I mad? Am I?"

func TestAESGCMEncodedEncrytor(t *testing.T) {
	primaryKey, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}

	secondaryKey, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("use primary key to encrypt", func(t *testing.T) {
		e := newAESGCMEncodedEncryptor(primaryKey, nil)

		encrypted, err := e.Encrypt(messageToEncrypt)
		if err != nil {
			t.Fatal(err)
		}

		decrypted, err := e.Decrypt(encrypted)
		if err != nil {
			t.Fatal(err)
		}

		if decrypted != messageToEncrypt {
			t.Fatal("unable to decrypt and get the original bytes")
		}
	})

	t.Run("use secondary key to encrypt", func(t *testing.T) {
		// Only load secondary key to encrypt
		e := newAESGCMEncodedEncryptor(secondaryKey, nil)
		encrypted, err := e.Encrypt(messageToEncrypt)
		if err != nil {
			t.Fatal(err)
		}

		// Then load both keys to decrypt
		e = newAESGCMEncodedEncryptor(primaryKey, secondaryKey)
		decrypted, err := e.Decrypt(encrypted)
		if err != nil {
			t.Fatal(err)
		}

		if decrypted != messageToEncrypt {
			t.Fatal("unable to decrypt and get the original bytes")
		}
	})

	isInSliceOnce := func(item string, slice []string) bool {
		found := 0
		for _, s := range slice {
			if item == s {
				found++
			}
		}

		return found == 1
	}

	t.Run("different messages should generate different outputs", func(t *testing.T) {
		e := newAESGCMEncodedEncryptor(primaryKey, nil)

		messages := []string{
			"This may or may",
			"This is not the same as that",
			"The end of that",
			"Plants and animals",
			"Snow, igloos, sunshine, unicords",
		}

		var crypts []string
		for _, m := range messages {
			encrypted, err := e.Encrypt(m)
			if err != nil {
				t.Fatal(err)
			}
			crypts = append(crypts, string(encrypted))
		}

		for _, c := range crypts {
			if !isInSliceOnce(c, crypts) {
				t.Fatalf("Duplicate encryption string: %v", c)
			}
		}
	})

	t.Run("same message generates different outputs every time", func(t *testing.T) {
		e := newAESGCMEncodedEncryptor(primaryKey, nil)

		var crypts []string
		for i := 0; i < 1000; i++ {
			encrypted, err := e.Encrypt(messageToEncrypt)
			if err != nil {
				t.Fatal(err)
			}
			crypts = append(crypts, encrypted)
		}

		for _, item := range crypts {
			if isInSliceOnce(item, crypts) == false {
				t.Fatalf("Duplicate encrypted string found")
			}
		}
	})
}

func TestAESGCMEncodedEncrytor_KeyHash(t *testing.T) {
	primaryKey, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}

	secondaryKey, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}

	e := newAESGCMEncodedEncryptor(primaryKey, secondaryKey)

	encryptorHash := e.PrimaryKeyHash()

	primaryKeyHash := sliceKeyHash(primaryKey)
	secondaryKeyHash := sliceKeyHash(secondaryKey)

	// SHOULD NOT equal secondaryKey hash
	if encryptorHash == secondaryKeyHash {
		t.Errorf("expected PrimaryKeyHash() %q != secondaryKeyHash %q", encryptorHash, secondaryKeyHash)
	}
	// SHOULD equal primaryKey hash
	if encryptorHash != primaryKeyHash {
		t.Errorf("expected PrimaryKeyHash() %q == PrimaryKeyHash() %q", encryptorHash, primaryKeyHash)
	}
}

func TestAESGCMEncodedEncrytor_BadKeysFailToDecrypt(t *testing.T) {
	key, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}

	e := newAESGCMEncodedEncryptor(key, nil)

	encrypted, err := e.Encrypt(messageToEncrypt)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := e.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	notTheSameKey, err := generateRandomAESKey()
	if err != nil {
		t.Fatal(err)
	}
	e2 := newAESGCMEncodedEncryptor(notTheSameKey, nil)

	decryptedAgain, err := e2.Decrypt(encrypted)
	// Not the same key will have different keyHash and effectively makes the decryption no-op.
	if err != ErrDecryptAttemptedButFailed {
		t.Fatalf("err: want %v but got %v", ErrDecryptAttemptedButFailed, err)
	}
	if decrypted == decryptedAgain {
		t.Fatal("Should not have been able to Decrypt string with an invalid secret key")
	}
}

func TestGatherKeys(t *testing.T) {
	tests := []struct {
		name             string
		data             []byte
		wantPrimaryKey   []byte
		wantSecondaryKey []byte
		wantErr          bool
	}{
		{
			name:             "base-case",
			data:             []byte("key123,key345"),
			wantPrimaryKey:   []byte("key123"),
			wantSecondaryKey: []byte("key345"),
			wantErr:          false,
		},
		{
			name:             "no key set",
			data:             []byte("key123"),
			wantPrimaryKey:   []byte("key123"),
			wantSecondaryKey: nil,
			wantErr:          false,
		},
		{
			name:             "3 key err case",
			data:             []byte("look mom, I am a key, me too"),
			wantPrimaryKey:   nil,
			wantSecondaryKey: nil,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrimaryKey, gotSecondaryKey, err := gatherKeys(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("gatherKeys() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !bytes.Equal(gotPrimaryKey, tt.wantPrimaryKey) {
				t.Errorf("gatherKeys() oOldKey = %s, want %s", gotSecondaryKey, tt.wantPrimaryKey)
			}
			if !bytes.Equal(gotSecondaryKey, tt.wantSecondaryKey) {
				t.Errorf("gathrKeys() gotPrimaryKey = %s, want %s", gotPrimaryKey, tt.wantSecondaryKey)
			}
		})
	}
}

func TestAESGCMEncodedEncrytor_RotateEncryption(t *testing.T) {
	tests := []struct {
		name           string
		primaryKey     []byte
		secondaryKey   []byte
		mockCiphertext string
		wantPlaintext  string
		wantErr        bool
	}{
		{
			name:          "base",
			primaryKey:    mustGenerateRandomAESKey(),
			secondaryKey:  mustGenerateRandomAESKey(),
			wantPlaintext: "this is a special string",
			wantErr:       false,
		},
		{
			name:           "failed_case",
			primaryKey:     mustGenerateRandomAESKey(),
			secondaryKey:   mustGenerateRandomAESKey(),
			mockCiphertext: "jdklasjdklsa$",
			wantPlaintext:  "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// pretend we originally use secondaryKey
			e2 := newAESGCMEncodedEncryptor(tt.secondaryKey, nil)
			ciphertext, err := e2.Encrypt(tt.wantPlaintext)
			if err != nil {
				t.Fatal(err)
			}

			if tt.mockCiphertext != "" {
				ciphertext = tt.mockCiphertext
			}

			// rotate to use new primary key
			e := newAESGCMEncodedEncryptor(tt.primaryKey, tt.secondaryKey)
			got, err := e.RotateEncryption(ciphertext)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RotateEncryption() error = %v, wantErr %v", err, tt.wantErr)
			}

			// we should always Encrypt with the primary key
			e3 := newAESGCMEncodedEncryptor(tt.primaryKey, nil)
			got, err = e3.Decrypt(got)
			if err != nil {
				t.Fatalf("RotateEncryption() unable to Decrypt with primary key %v", err)
			}

			if diff := cmp.Diff(tt.wantPlaintext, got); diff != "" {
				t.Fatalf("Mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAESGCMEncodedEncrytor_Decrypt_Plaintext(t *testing.T) {
	tests := []struct {
		name          string
		primaryKey    []byte
		secondaryKey  []byte
		encryptInTest bool
		ciphertext    string
		wantPlaintext string
		wantErr       bool
	}{

		{
			name:          "base",
			primaryKey:    mustGenerateRandomAESKey(),
			secondaryKey:  mustGenerateRandomAESKey(),
			encryptInTest: true,
			ciphertext:    "VerySpecialString",
			wantPlaintext: "VerySpecialString",
			wantErr:       false,
		},
		{
			name:          "unencrypted ciphertext",
			primaryKey:    mustGenerateRandomAESKey(),
			secondaryKey:  mustGenerateRandomAESKey(),
			encryptInTest: false,
			ciphertext:    "Non-encrypted string",
			wantPlaintext: "Non-encrypted string",
			wantErr:       false,
		},
		{
			// this occurs when a token contains the `separator`
			name:          "unencrypted ciphertext with separator",
			primaryKey:    mustGenerateRandomAESKey(),
			secondaryKey:  mustGenerateRandomAESKey(),
			encryptInTest: false,
			ciphertext:    "VeryBadString" + separator,
			wantPlaintext: "",
			wantErr:       true,
		},
		{
			name:          "single key",
			primaryKey:    mustGenerateRandomAESKey(),
			secondaryKey:  nil,
			encryptInTest: false,
			ciphertext:    messageToEncrypt,
			wantPlaintext: messageToEncrypt,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var e encryptor
			if tt.primaryKey != nil && tt.secondaryKey != nil {
				e = newAESGCMEncodedEncryptor(tt.primaryKey, tt.secondaryKey)
			} else if tt.primaryKey != nil {
				e = newAESGCMEncodedEncryptor(tt.primaryKey, nil)
			} else if tt.secondaryKey != nil {
				e = newAESGCMEncodedEncryptor(tt.secondaryKey, nil)
			} else {
				t.Fatal("must have a non-nil key")
			}

			var err error
			if tt.encryptInTest {
				tt.ciphertext, err = e.Encrypt(tt.ciphertext)
				if err != nil {
					t.Fatal(err)
				}
			}

			gotPlaintext, err := e.Decrypt(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPlaintext != tt.wantPlaintext {
				t.Errorf("Decrypt() gotPlaintext = %q, want %q", gotPlaintext, tt.wantPlaintext)
			}
		})
	}
}

func TestNoOpEncryptor(t *testing.T) {
	t.Run("should do nothing when encrypt and decrypt", func(t *testing.T) {
		e := noOpEncryptor{}
		encString, err := e.Encrypt(messageToEncrypt)
		if err != nil {
			t.Fatalf("Received error when code path should be nil, but got %v", err)
		}
		if encString != messageToEncrypt {
			t.Fatalf("Received encrypted string, expected unencrypted")
		}
		decString, err := e.Decrypt(encString)
		if err != nil {
			t.Fatalf("Received error when code path should be nil, but got %v", err)
		}
		if decString != messageToEncrypt {
			t.Fatalf("Received encrypted string, expected unencrypted")
		}
	})

	t.Run("shouldn't have key hashes", func(t *testing.T) {
		e := noOpEncryptor{}
		if e.PrimaryKeyHash() != "" && e.SecondaryKeyHash() != "" {
			t.Fatal("noop encryptor shouldn't have key hashes")
		}
	})
}
