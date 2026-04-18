package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestSignAndParse_RoundTrip(t *testing.T) {
	secret := "this-is-a-long-enough-secret-32ch!!"
	id := uuid.New()
	tok, err := SignUserToken(secret, id, "alice", "admin", 24)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := ParseUserToken(secret, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Username != "alice" || claims.Role != "admin" || claims.Subject != id.String() {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParse_RejectsNoneAlg(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &UserClaims{Username: "x"})
	raw, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if _, err := ParseUserToken("any-secret", raw); err == nil {
		t.Fatalf("expected error on alg=none, got nil")
	}
}

func TestParse_RejectsRS256Tampering(t *testing.T) {
	// 构造一个 RS256 header 的 token；ParseUserToken 应在 keyFunc 中断言失败，
	// 防止把 HMAC secret 当作 RSA 公钥对待。
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &UserClaims{Username: "x"})
	// 用错误的 key 让 SignedString 产生合法 header；我们只关心 Parse 阶段。
	// 直接构造 header.claims. 签名乱填，Parse 会先进 keyFunc 检查算法即失败。
	raw := token.Raw // Raw is empty; build via encoded parts:
	if raw == "" {
		// 简易伪造：HS256 的 token 改 header alg 字段
		validHS, err := SignUserToken("s-32s-32s-32s-32s-32s-32s-32ss!!", uuid.New(), "x", "user", 1)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		raw = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." + splitPayloadSig(validHS)
	}
	if _, err := ParseUserToken("s-32s-32s-32s-32s-32s-32s-32ss!!", raw); err == nil {
		t.Fatalf("expected rejection of RS256-header token")
	}
}

func splitPayloadSig(token string) string {
	// 返回 token 中 header 之外的部分，以便拼接一个伪造的 header。
	for i, c := range token {
		if c == '.' {
			return token[i+1:]
		}
	}
	return token
}
