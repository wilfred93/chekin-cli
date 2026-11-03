package chekin

import (
	"encoding/base64"
	"fmt"
	"time"
)

// MakeTestToken builds an unsigned JWT-shaped token for tests.
func MakeTestToken(username string, exp time.Time) string {
	enc := base64.RawURLEncoding.EncodeToString
	payload := fmt.Sprintf(`{"username":%q,"exp":%d}`, username, exp.Unix())
	return enc([]byte(`{"typ":"JWT","alg":"none"}`)) + "." + enc([]byte(payload)) + ".sig"
}
