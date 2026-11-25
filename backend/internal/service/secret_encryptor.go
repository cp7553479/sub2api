package service

// SecretEncryptor protects provider credentials and other stored business secrets.
type SecretEncryptor interface {
 Encrypt(string)(string,error)
 Decrypt(string)(string,error)
}
