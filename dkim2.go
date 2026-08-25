package main

/*
 #cgo CFLAGS: -I/opt/halon/include
 #cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-all
 #include <HalonMTA.h>
 #include <openssl/evp.h>
 #include <openssl/pem.h>
 #include <openssl/bio.h>
 #include <stdio.h>
 #include <stdlib.h>

 static inline size_t read_file(FILE *fp, void *buf, size_t n) {
     return fread(buf, 1, n, fp);
 }
 static inline long bio_get_mem_data(BIO *b, char **pp) {
     return BIO_get_mem_data(b, pp);
 }
*/
import "C"

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"
	"unsafe"

	"go.turscar.ie/dkim2"
)

func main() {}

func GetArgumentAsJSON(args *C.HalonHSLArguments, pos uint64, required bool) (string, error) {
	var x = C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if x == nil {
		if required {
			return "", fmt.Errorf("missing argument at position %d", pos)
		} else {
			return "", nil
		}
	}
	var y *C.char
	z := C.HalonMTA_hsl_value_to_json(x, &y, nil)
	defer C.free(unsafe.Pointer(y))
	if z {
		return C.GoString(y), nil
	} else {
		return "", fmt.Errorf("invalid argument at position %d", pos)
	}
}

func GetArgumentAsString(args *C.HalonHSLArguments, pos uint64, required bool) (string, error) {
	var x = C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if x == nil {
		if required {
			return "", fmt.Errorf("missing argument at position %d", pos)
		} else {
			return "", nil
		}
	}
	var y *C.char
	if C.HalonMTA_hsl_value_get(x, C.HALONMTA_HSL_TYPE_STRING, unsafe.Pointer(&y), nil) {
		return C.GoString(y), nil
	} else {
		return "", fmt.Errorf("invalid argument at position %d", pos)
	}
}

func GetArgumentAsKey(args *C.HalonHSLArguments, pos uint64) (string, error) {
	arg := C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if arg == nil {
		return "", fmt.Errorf("missing argument at position %d", pos)
	}

	var pkey *C.EVP_PKEY
	if !bool(C.HalonMTA_hsl_value_get(arg, C.HALONMTA_HSL_TYPE_PRIVATEKEY, unsafe.Pointer(&pkey), nil)) {
		return "", fmt.Errorf("argument %d is not a private key", pos)
	}
	if pkey == nil {
		return "", fmt.Errorf("private key argument is nil")
	}

	bio := C.BIO_new(C.BIO_s_mem())
	if bio == nil {
		return "", fmt.Errorf("failed to allocate BIO")
	}
	defer C.BIO_free(bio)

	if C.PEM_write_bio_PrivateKey(bio, pkey, nil, nil, 0, nil, nil) != 1 {
		return "", fmt.Errorf("failed to write private key to PEM")
	}

	var data *C.char
	n := C.bio_get_mem_data(bio, &data)
	if n <= 0 || data == nil {
		return "", fmt.Errorf("failed to read PEM data from BIO")
	}

	return C.GoStringN(data, C.int(n)), nil
}

func SetException(hhc *C.HalonHSLContext, msg string) {
	x := C.CString(msg)
	y := unsafe.Pointer(x)
	defer C.free(y)
	exception := C.HalonMTA_hsl_throw(hhc)
	C.HalonMTA_hsl_value_set(exception, C.HALONMTA_HSL_TYPE_EXCEPTION, y, 0)
}

func SetReturnValueToAny(ret *C.HalonHSLValue, val interface{}) error {
	x, err := json.Marshal(val)
	if err != nil {
		return err
	}
	y := C.CString(string(x))
	defer C.free(unsafe.Pointer(y))
	var z *C.char
	if !(C.HalonMTA_hsl_value_from_json(ret, y, &z, nil)) {
		if z != nil {
			err = errors.New(C.GoString(z))
			C.free(unsafe.Pointer(z))
		} else {
			err = errors.New("failed to parse return value")
		}
		return err
	}
	return nil
}

//export Halon_version
func Halon_version() C.int {
	return C.HALONMTA_PLUGIN_VERSION
}

//export Halon_hsl_register
func Halon_hsl_register(hhrc *C.HalonHSLRegisterContext) C.bool {
	dkim2_verify_cs := C.CString("dkim2_verify")
	C.HalonMTA_hsl_module_register_function(hhrc, dkim2_verify_cs, nil)
	dkim2_sign_cs := C.CString("dkim2_sign")
	C.HalonMTA_hsl_module_register_function(hhrc, dkim2_sign_cs, nil)
	return true
}

type fileReader struct {
	fp *C.FILE
}

type DKIM2VerifyOptions struct {
	IgnoreTimestamp bool `json:"ignoretimestamp"`
}

type DKIM2SignOptions struct {
	Nonce        string `json:"nonce"`
	Timestamp    int64  `json:"timestamp"`
	Exploded     bool   `json:"exploded"`
	DoNotExplode bool   `json:"donotexplode"`
	DoNotModify  bool   `json:"donotmodify"`
	Feedback     bool   `json:"feedback"`
}

type DKIM2VerifyReturnValue struct {
	Result *dkim2.VerificationState `json:"result"`
	Error  string                   `json:"error"`
}

type DKIM2SignReturnValue struct {
	Result  bool     `json:"result"`
	Error   string   `json:"error"`
	Headers []string `json:"headers"`
}

func (r *fileReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	n := C.read_file(
		r.fp,
		unsafe.Pointer(&p[0]),
		C.size_t(len(p)),
	)

	if n == 0 {
		return 0, io.EOF
	}

	return int(n), nil
}

func unmarshalOptions(data string, target any) error {
	raw := json.RawMessage(data)

	var array []json.RawMessage
	if err := json.Unmarshal(raw, &array); err == nil {
		if len(array) == 0 {
			return nil
		}
		return fmt.Errorf("options must be an object")
	}

	return json.Unmarshal(raw, target)
}

func getFileReader(args *C.HalonHSLArguments, pos uint64) (*fileReader, error) {
	arg := C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if arg == nil {
		return nil, fmt.Errorf("missing mail file argument")
	}

	var fp *C.FILE
	if !bool(C.HalonMTA_hsl_value_get(
		arg,
		C.HALONMTA_HSL_TYPE_FILE,
		unsafe.Pointer(&fp),
		nil,
	)) {
		return nil, fmt.Errorf("argument %d is not a file", pos)
	}
	if fp == nil {
		return nil, fmt.Errorf("file argument is nil")
	}
	if C.fseek(fp, 0, C.SEEK_SET) != 0 {
		return nil, fmt.Errorf("failed to seek mail file")
	}

	return &fileReader{fp: fp}, nil
}

func parseStringArrayArgument(args *C.HalonHSLArguments, pos uint64) ([]string, error) {
	value, err := GetArgumentAsJSON(args, pos, true)
	if err != nil {
		return nil, err
	}
	var result []string
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("invalid argument at position %d: %w", pos, err)
	}
	return result, nil
}

func parsePrivateKey(value string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, fmt.Errorf("private key is not PEM encoded")
	}

	var key any
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported PEM block %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	switch key := key.(type) {
	case *rsa.PrivateKey:
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("invalid RSA private key: %w", err)
		}
		return key, nil
	case ed25519.PrivateKey:
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
}

//export dkim2_sign
func dkim2_sign(hhc *C.HalonHSLContext, args *C.HalonHSLArguments, ret *C.HalonHSLValue) {
	reader, err := getFileReader(args, 0)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	mailFrom, err := GetArgumentAsString(args, 1, true)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	rcptTo, err := parseStringArrayArgument(args, 2)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	selector, err := GetArgumentAsString(args, 3, true)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	domain, err := GetArgumentAsString(args, 4, true)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	privateKeyPEM, err := GetArgumentAsKey(args, 5)
	if err != nil {
		privateKeyPEM, err = GetArgumentAsString(args, 5, true)
		if err != nil {
			SetException(hhc, err.Error())
			return
		}
	}
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}

	var opts DKIM2SignOptions
	optionsJSON, err := GetArgumentAsJSON(args, 6, false)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	if optionsJSON != "" {
		if err := unmarshalOptions(optionsJSON, &opts); err != nil {
			SetException(hhc, fmt.Sprintf("invalid signing options: %v", err))
			return
		}
	}
	if strings.TrimSpace(selector) == "" {
		SetException(hhc, "selector must not be empty")
		return
	}
	if strings.TrimSpace(domain) == "" {
		SetException(hhc, "domain must not be empty")
		return
	}

	message, err := mail.ReadMessage(reader)
	if err != nil {
		SetException(hhc, fmt.Sprintf("failed to parse mail: %v", err))
		return
	}

	options := dkim2.SignOptions{
		Nonce:        opts.Nonce,
		Timestamp:    opts.Timestamp,
		Domain:       domain,
		Keys:         []dkim2.SigningKey{{Selector: selector, Signer: privateKey}},
		Exploded:     opts.Exploded,
		DoNotExplode: opts.DoNotExplode,
		DoNotModify:  opts.DoNotModify,
		Feedback:     opts.Feedback,
		MailFrom:     mailFrom,
		RcptTo:       rcptTo,
	}

	headers, err := dkim2.SignMessage(message, options)

	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	value := DKIM2SignReturnValue{
		Result:  err == nil,
		Headers: headers,
	}
	if value.Headers == nil {
		value.Headers = []string{}
	}
	if err := SetReturnValueToAny(ret, value); err != nil {
		SetException(hhc, err.Error())
	}
}

//export dkim2_verify
func dkim2_verify(hhc *C.HalonHSLContext, args *C.HalonHSLArguments, ret *C.HalonHSLValue) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader, err := getFileReader(args, 0)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}

	mailFrom, err := GetArgumentAsString(args, 1, true)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}

	rcptTo, err := parseStringArrayArgument(args, 2)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}

	var opts DKIM2VerifyOptions
	optionsJSON, err := GetArgumentAsJSON(args, 3, false)
	if err != nil {
		SetException(hhc, err.Error())
		return
	}
	if optionsJSON != "" {
		if err := unmarshalOptions(optionsJSON, &opts); err != nil {
			SetException(hhc, fmt.Sprintf("invalid verify options: %v", err))
			return
		}
	}

	res := dkim2.Verify(ctx, reader, dkim2.VerifyOptions{
		MailFrom:        mailFrom,
		RcptTo:          rcptTo,
		IgnoreTimestamp: opts.IgnoreTimestamp,
	})

	dkim2State := res.State()
	result := &dkim2State

	if _, ok := errors.AsType[dkim2.ErrSignatureMissing](res.Err); ok {
		result = nil
	}

	errorMessage := ""
	if res.Err != nil {
		errorMessage = res.AuthenticationResult()
	}

	value := DKIM2VerifyReturnValue{
		Result: result,
		Error:  errorMessage,
	}

	SetReturnValueToAny(ret, value)
}
