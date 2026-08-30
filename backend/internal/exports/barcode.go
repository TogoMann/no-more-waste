package exports

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"math/rand"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
)

func GenerateBarcodeValue() string {
	source := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("NMW%010d", source.Int63n(9999999999))
}

func BarcodePNGBase64(value string) (string, error) {
	encoded, err := code128.Encode(value)
	if err != nil {
		return "", err
	}
	scaled, err := barcode.Scale(encoded, 300, 100)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, scaled); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
