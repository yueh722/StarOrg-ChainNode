package security

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/mathutil"
)

type Auth struct {
	secret []byte
}

func NewAuth(secret []byte) *Auth {
	return &Auth{secret: secret}
}

var SharedSecret = []byte{
	182, 19, 122, 212, 127, 206, 37, 103,
	54, 152, 168, 175, 98, 176, 158, 20,
	149, 101, 192, 23, 227, 145, 167, 182,
	112, 39, 121, 44, 42, 200, 135, 170,
}

// Token 有效期窗口（±5 分钟）
const TimeWindow = 5 * time.Minute

type clientIDKey struct{}

func WithClientID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, clientIDKey{}, id)
}

func GetClientID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(clientIDKey{}).(string)
	return id, ok
}

func GenerateToken(clientID string, secret []byte) string {
	sig := Signature(time.Now().Unix(), clientID, secret)
	return fmt.Sprintf("%d:%x", time.Now().Unix(), sig)
}

func RandString(n int32) string {
	b := make([]byte, n)
	rand.Read(b)
	return (fmt.Sprintf("%x", b))
}

func Signature(ts int64, clientID string, secret []byte) []byte {
	mac := hmac.New(sha256.New, (secret))
	mac.Write([]byte(clientID))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	return mac.Sum(nil)
}

func (a *Auth) validateToken(clientID, token string) bool {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return false
	}

	tsStr, sig := parts[0], parts[1]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}

	// check time replay
	now := time.Now().Unix()
	if mathutil.Abs(now-ts) > int64(TimeWindow.Seconds()) {
		return false
	}

	expectedSig := fmt.Sprintf("%x", Signature(ts, clientID, a.secret))

	// check signature
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			http.Error(w, "missing X-Client-ID", http.StatusUnauthorized)
			return
		}

		//zlog.Info("validateToken clientID: ", clientID)

		if !a.validateToken(clientID, token) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := WithClientID(r.Context(), clientID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
