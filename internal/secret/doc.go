/*
Package secrets provides encryption at rest to a Sourcegraph deployment using AES-GCM. It requires an initial secret
to be passed in via a secretfile or an env var in order to provide this encryption.

The intended usage is at the database layer when access occurs. Determined secret data should be passed through
EncryptBytes before being posted to that database. DecryptBytes should then be used to provide the plaintext.

The key order matters in the env var or secretfile. It is expected to be a comma separated string with the primaryKey
being used for encryption
		"primaryKey,secondaryKey"
The secondaryKey is only required when key rotation is desired and is not required to perform encryption.

*/
package secret
