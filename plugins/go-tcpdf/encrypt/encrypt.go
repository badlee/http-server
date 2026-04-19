// Package encrypt provides PDF document encryption (RC4-40, RC4-128, AES-128, AES-256).
// Ported from tc-lib-pdf-encrypt (PHP) by Nicola Asuni.
package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

// EncryptionMode selects the algorithm.
type EncryptionMode int

const (
	EncNone    EncryptionMode = 0
	EncRC4_40  EncryptionMode = 1  // RC4 40-bit (PDF 1.1)
	EncRC4_128 EncryptionMode = 2  // RC4 128-bit (PDF 1.4)
	EncAES_128 EncryptionMode = 3  // AES 128-bit (PDF 1.6)
	EncAES_256 EncryptionMode = 4  // AES 256-bit (PDF 1.7 ext3)
)

// Permissions bit flags (Table 22 in PDF 1.7 spec).
const (
	PermPrint          = 1 << 2  // bit 3
	PermModify         = 1 << 3  // bit 4
	PermCopy           = 1 << 4  // bit 5
	PermAnnot          = 1 << 5  // bit 6
	PermFillForms      = 1 << 8  // bit 9
	PermExtract        = 1 << 9  // bit 10
	PermAssemble       = 1 << 10 // bit 11
	PermPrintHighQual  = 1 << 11 // bit 12
	PermAll            = PermPrint | PermModify | PermCopy | PermAnnot |
		PermFillForms | PermExtract | PermAssemble | PermPrintHighQual
)

// Config holds encryption configuration.
type Config struct {
	Mode          EncryptionMode
	UserPassword  string
	OwnerPassword string
	Permissions   int32
}

// Encrypt manages PDF encryption state.
type Encrypt struct {
	cfg         Config
	encKey      []byte
	userKey     []byte // /U entry
	ownerKey    []byte // /O entry
	fileID      []byte // document ID
	ue          []byte // /UE (AES-256)
	oe          []byte // /OE (AES-256)
	permsEntry  []byte // /Perms (AES-256)
	objNum      int
	genNum      int
}

// New creates a new Encrypt instance.
func New(cfg Config) (*Encrypt, error) {
	e := &Encrypt{cfg: cfg}
	id := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return nil, fmt.Errorf("encrypt: failed to generate file ID: %w", err)
	}
	e.fileID = id
	if err := e.computeKeys(); err != nil {
		return nil, err
	}
	return e, nil
}

// Disabled returns true when no encryption is configured.
func (e *Encrypt) Disabled() bool {
	return e == nil || e.cfg.Mode == EncNone
}

// FileID returns the document ID bytes.
func (e *Encrypt) FileID() []byte { return e.fileID }

// UserKey returns the /U dictionary entry.
func (e *Encrypt) UserKey() []byte { return e.userKey }

// OwnerKey returns the /O dictionary entry.
func (e *Encrypt) OwnerKey() []byte { return e.ownerKey }

// UE returns the /UE dictionary entry (AES-256 only).
func (e *Encrypt) UE() []byte { return e.ue }

// OE returns the /OE dictionary entry (AES-256 only).
func (e *Encrypt) OE() []byte { return e.oe }

// Perms returns the /Perms dictionary entry (AES-256 only).
func (e *Encrypt) Perms() []byte { return e.permsEntry }

// KeyLength returns the encryption key length in bits.
func (e *Encrypt) KeyLength() int {
	switch e.cfg.Mode {
	case EncRC4_40:
		return 40
	case EncRC4_128:
		return 128
	case EncAES_128:
		return 128
	case EncAES_256:
		return 256
	}
	return 0
}

// Revision returns the PDF encryption revision number.
func (e *Encrypt) Revision() int {
	switch e.cfg.Mode {
	case EncRC4_40:
		return 2
	case EncRC4_128:
		return 3
	case EncAES_128:
		return 4
	case EncAES_256:
		return 6
	}
	return 0
}

// Version returns the /V value for the encryption dictionary.
func (e *Encrypt) Version() int {
	switch e.cfg.Mode {
	case EncRC4_40:
		return 1
	case EncRC4_128:
		return 2
	case EncAES_128:
		return 4
	case EncAES_256:
		return 5
	}
	return 0
}

// SetObjNum sets the current object/generation number for stream encryption.
func (e *Encrypt) SetObjNum(objNum, genNum int) {
	e.objNum = objNum
	e.genNum = genNum
}

// EncryptBytes encrypts a byte slice for the current object.
func (e *Encrypt) EncryptBytes(data []byte) ([]byte, error) {
	if e.Disabled() {
		return data, nil
	}
	switch e.cfg.Mode {
	case EncRC4_40, EncRC4_128:
		return e.encryptRC4(data)
	case EncAES_128:
		return e.encryptAES128(data)
	case EncAES_256:
		return e.encryptAES256(data)
	}
	return data, nil
}

// EncryptString encrypts a PDF string for the current object.
func (e *Encrypt) EncryptString(s string) ([]byte, error) {
	return e.EncryptBytes([]byte(s))
}

// ---- Key derivation -----------------------------------------------------

func (e *Encrypt) computeKeys() error {
	switch e.cfg.Mode {
	case EncRC4_40, EncRC4_128:
		return e.computeRC4Keys()
	case EncAES_128:
		return e.computeAES128Keys()
	case EncAES_256:
		return e.computeAES256Keys()
	}
	return nil
}

// Padding used by PDF RC4 key generation (PDF spec §7.6.3.3).
var padding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

func padPassword(pw string) []byte {
	b := []byte(pw)
	if len(b) >= 32 {
		return b[:32]
	}
	result := make([]byte, 32)
	copy(result, b)
	copy(result[len(b):], padding)
	return result
}

func (e *Encrypt) computeRC4Keys() error {
	keyLen := 5 // 40-bit
	if e.cfg.Mode == EncRC4_128 {
		keyLen = 16 // 128-bit
	}

	// Compute owner key O
	h := md5.New()
	h.Write(padPassword(e.cfg.OwnerPassword))
	oHash := h.Sum(nil)
	if e.cfg.Mode == EncRC4_128 {
		for i := 0; i < 50; i++ {
			h.Reset()
			h.Write(oHash[:keyLen])
			oHash = h.Sum(nil)
		}
	}
	ownerKey := oHash[:keyLen]
	c, err := rc4.NewCipher(ownerKey)
	if err != nil {
		return err
	}
	oEntry := make([]byte, 32)
	c.XORKeyStream(oEntry, padPassword(e.cfg.UserPassword))
	if e.cfg.Mode == EncRC4_128 {
		tmp := make([]byte, 32)
		for i := 1; i <= 19; i++ {
			k := make([]byte, keyLen)
			for j := range k {
				k[j] = ownerKey[j] ^ byte(i)
			}
			c2, _ := rc4.NewCipher(k)
			c2.XORKeyStream(tmp, oEntry)
			copy(oEntry, tmp)
		}
	}
	e.ownerKey = oEntry

	// Compute encryption key
	h.Reset()
	h.Write(padPassword(e.cfg.UserPassword))
	h.Write(oEntry)
	perm := make([]byte, 4)
	binary.LittleEndian.PutUint32(perm, uint32(e.cfg.Permissions))
	h.Write(perm)
	h.Write(e.fileID)
	encKey := h.Sum(nil)[:keyLen]
	if e.cfg.Mode == EncRC4_128 {
		for i := 0; i < 50; i++ {
			h.Reset()
			h.Write(encKey)
			encKey = h.Sum(nil)[:keyLen]
		}
	}
	e.encKey = encKey

	// Compute user key U
	h.Reset()
	h.Write(padding)
	h.Write(e.fileID)
	uHash := h.Sum(nil)
	c2, _ := rc4.NewCipher(encKey)
	uEntry := make([]byte, 32)
	if e.cfg.Mode == EncRC4_128 {
		c2.XORKeyStream(uEntry, uHash)
		tmp := make([]byte, 16)
		for i := 1; i <= 19; i++ {
			k := make([]byte, keyLen)
			for j := range k {
				k[j] = encKey[j] ^ byte(i)
			}
			c3, _ := rc4.NewCipher(k)
			c3.XORKeyStream(tmp, uEntry[:16])
			copy(uEntry[:16], tmp)
		}
	} else {
		c2.XORKeyStream(uEntry, padding)
	}
	e.userKey = uEntry
	return nil
}

func (e *Encrypt) computeAES128Keys() error {
	// AES-128 uses same key derivation as RC4-128 (PDF Rev 4)
	e.cfg.Mode = EncRC4_128
	if err := e.computeRC4Keys(); err != nil {
		return err
	}
	e.cfg.Mode = EncAES_128
	return nil
}

func (e *Encrypt) computeAES256Keys() error {
	// AES-256 (Rev 6) — simplified implementation
	// User validation salt and key salt
	uVS := make([]byte, 8)
	uKS := make([]byte, 8)
	io.ReadFull(rand.Reader, uVS)
	io.ReadFull(rand.Reader, uKS)

	// File encryption key (32 random bytes)
	fileKey := make([]byte, 32)
	io.ReadFull(rand.Reader, fileKey)
	e.encKey = fileKey

	// User hash: SHA256(password + uVS)
	upw := []byte(e.cfg.UserPassword)
	h := sha256.New()
	h.Write(upw)
	h.Write(uVS)
	uHash := h.Sum(nil)
	// /U = uHash(32) + uVS(8) + uKS(8)
	e.userKey = append(append(uHash, uVS...), uKS...)

	// /UE = AES256(SHA256(upw + uKS), fileKey)
	h.Reset()
	h.Write(upw)
	h.Write(uKS)
	uKeyKey := h.Sum(nil)
	ue, _ := aesWrap(uKeyKey, fileKey)
	e.ue = ue

	// Owner validation salt and key salt
	oVS := make([]byte, 8)
	oKS := make([]byte, 8)
	io.ReadFull(rand.Reader, oVS)
	io.ReadFull(rand.Reader, oKS)

	opw := []byte(e.cfg.OwnerPassword)
	h.Reset()
	h.Write(opw)
	h.Write(oVS)
	h.Write(e.userKey)
	oHash := h.Sum(nil)
	e.ownerKey = append(append(oHash, oVS...), oKS...)

	h.Reset()
	h.Write(opw)
	h.Write(oKS)
	h.Write(e.userKey)
	oKeyKey := h.Sum(nil)
	oe, _ := aesWrap(oKeyKey, fileKey)
	e.oe = oe

	// /Perms
	permsPlain := make([]byte, 16)
	binary.LittleEndian.PutUint32(permsPlain, uint32(e.cfg.Permissions))
	permsPlain[4] = 0xFF; permsPlain[5] = 0xFF; permsPlain[6] = 0xFF; permsPlain[7] = 0xFF
	permsPlain[8] = 'T' // EncryptMetadata = true
	permsPlain[9] = 'a'; permsPlain[10] = 'd'; permsPlain[11] = 'b'
	io.ReadFull(rand.Reader, permsPlain[12:])
	block, _ := aes.NewCipher(fileKey)
	permsEnc := make([]byte, 16)
	block.Encrypt(permsEnc, permsPlain)
	e.permsEntry = permsEnc
	return nil
}

// aesWrap encrypts key with kek using AES-CBC with zero IV (simplified RFC3394-like).
func aesWrap(kek, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize) // zero IV
	mode := cipher.NewCBCEncrypter(block, iv)
	// pad key to multiple of block size
	padded := key
	if rem := len(padded) % aes.BlockSize; rem != 0 {
		padded = append(padded, make([]byte, aes.BlockSize-rem)...)
	}
	out := make([]byte, len(padded))
	mode.CryptBlocks(out, padded)
	return out, nil
}

// ---- Stream encryption --------------------------------------------------

func (e *Encrypt) objectKey() []byte {
	if e.cfg.Mode == EncAES_256 {
		return e.encKey // AES-256 uses file key directly
	}
	// Extend key with object number and generation number (PDF §7.6.3.3 step 1-2)
	tmp := make([]byte, len(e.encKey)+5)
	copy(tmp, e.encKey)
	tmp[len(e.encKey)] = byte(e.objNum)
	tmp[len(e.encKey)+1] = byte(e.objNum >> 8)
	tmp[len(e.encKey)+2] = byte(e.objNum >> 16)
	tmp[len(e.encKey)+3] = byte(e.genNum)
	tmp[len(e.encKey)+4] = byte(e.genNum >> 8)
	if e.cfg.Mode == EncAES_128 {
		tmp = append(tmp, 0x73, 0x41, 0x6C, 0x54) // "sAlT"
	}
	h := md5.Sum(tmp)
	keyLen := len(e.encKey) + 5
	if keyLen > 16 {
		keyLen = 16
	}
	return h[:keyLen]
}

func (e *Encrypt) encryptRC4(data []byte) ([]byte, error) {
	key := e.objectKey()
	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return out, nil
}

func (e *Encrypt) encryptAES128(data []byte) ([]byte, error) {
	key := e.objectKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// Pad to block size (PKCS7)
	padLen := aes.BlockSize - len(data)%aes.BlockSize
	padded := append(data, make([]byte, padLen)...)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	// Random IV prepended
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)
	out := make([]byte, aes.BlockSize+len(padded))
	copy(out, iv)
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[aes.BlockSize:], padded)
	return out, nil
}

func (e *Encrypt) encryptAES256(data []byte) ([]byte, error) {
	// AES-256-CBC with random IV
	block, err := aes.NewCipher(e.encKey)
	if err != nil {
		return nil, err
	}
	padLen := aes.BlockSize - len(data)%aes.BlockSize
	padded := append(data, make([]byte, padLen)...)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)
	out := make([]byte, aes.BlockSize+len(padded))
	copy(out, iv)
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[aes.BlockSize:], padded)
	return out, nil
}
