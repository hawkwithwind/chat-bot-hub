package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"
	"database/sql"

	mt "github.com/mitchellh/mapstructure"
)

func CheckSum(src []byte) []byte {
	h := sha256.New()
	h.Write(src)
	return h.Sum(nil)
}

func HexString(src []byte) string {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	return string(dst)
}

func PasswordCheckSum(pass string) string {
	return HexString(CheckSum([]byte(pass)))
}

func DecodeMap(src interface{}, target interface{}) error {
	config := &mt.DecoderConfig{
		DecodeHook: func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
			if t == reflect.TypeOf(time.Time{}) && f == reflect.TypeOf("") {
				return time.Parse(time.RFC3339, data.(string))
			} else if t == reflect.TypeOf(JSONTime{}) && f == reflect.TypeOf("") {
				if tt, err := time.Parse(time.RFC3339, data.(string)); err == nil {
					return JSONTime{tt}, nil
				} else {
					return nil, err
				}
			}

			return data, nil
		},
		Metadata: nil,
		Result:   target,
	}

	decoder, err := mt.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(src)
}

func StringNull(str string, defaultValue string) sql.NullString {
	if str == defaultValue {
		return sql.NullString {
			String: "",
			Valid: false,
		}
	} else {
		return sql.NullString {
			String: str,
			Valid: true,
		}
	}
}
