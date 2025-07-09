package help

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"math"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// IsExist 判断文件是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {

		return true
	}

	if os.IsExist(err) {

		return true
	}

	return false
}

func GetEnv(key string) string {

	return os.Getenv(key)
}

func EpusdtSign(data map[string]interface{}, token string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == "signature" {

			continue
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)
	var sign strings.Builder
	for _, k := range keys {
		v := data[k]
		if v == nil || v == "" {

			continue
		}

		sign.WriteString(k)
		sign.WriteString("=")
		sign.WriteString(fmt.Sprintf("%v", v))
		sign.WriteString("&")
	}

	signString := strings.TrimRight(sign.String(), "&")

	return Md5String(signString + token)
}

func GenerateTradeId() (string, error) {
	var defaultAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	return gonanoid.Generate(defaultAlphabet, 18)
}

func Md5String(text string) string {

	return fmt.Sprintf("%x", md5.Sum([]byte(text)))
}

func Ec(str string) string {
	escapeChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	for _, char := range escapeChars {
		str = strings.ReplaceAll(str, char, "\\"+char)
	}

	return str
}

func IsNumber(s string) bool {
	match, err := regexp.MatchString(`^\d+\.?\d*$`, s)

	return match && err == nil
}

func IsValidTronAddress(address string) bool {
	match, err := regexp.MatchString(`^T[a-zA-Z0-9]{33}$`, address)

	return match && err == nil
}

func IsValidEvmAddress(address string) bool {
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {

		return false
	}

	addrWithoutPrefix := address[2:]
	if _, err := hex.DecodeString(addrWithoutPrefix); err != nil {

		return false
	}

	return true
}

func IsValidSolanaAddress(address string) bool {
	data := base58.Decode(address)

	return len(data) == 32
}

func MaskAddress(address string) string {
	if len(address) <= 20 {

		return address
	}

	return address[:8] + " ***** " + address[len(address)-10:]
}

func MaskAddress2(address string) string {
	if len(address) <= 20 {

		return address
	}

	return "*** " + address[len(address)-8:]
}

func MaskHash(hash string) string {
	if len(hash) <= 20 {

		return hash
	}

	return hash[:6] + " ***** " + hash[len(hash)-8:]
}

func CalcNextNotifyTime(base time.Time, num int) time.Time {

	return base.Add(time.Minute * time.Duration(math.Pow(2, float64(num))))
}

func HexStr2Int(str string) *big.Int {
	var n = new(big.Int)
	var val = strings.TrimLeft(strings.TrimPrefix(str, "0x"), "0")

	n.SetString(val, 16)

	return n
}

func InStrings(str string, list []string) bool {
	for _, item := range list {
		if item == str {

			return true
		}
	}

	return false
}
