package jwt

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	dgrijalva "github.com/vtolstov/jwt-go"
)

var benchClaims = &Claims{
	Registered: Registered{
		Issuer: "benchmark",
		Issued: NewNumericTime(time.Now()),
	},
}

var dgrijalvaClaims = dgrijalva.StandardClaims{
	Issuer:   benchClaims.Issuer,
	IssuedAt: int64(*benchClaims.Issued),
}

func BenchmarkECDSA(b *testing.B) {
	tests := []struct {
		key *ecdsa.PrivateKey
		alg string
	}{
		{testKeyEC256, ES256},
		{testKeyEC384, ES384},
		{testKeyEC521, ES512},
	}

	for _, test := range tests {
		b.Run("sign-"+test.alg, func(b *testing.B) {
			var tokenLen int
			for i := 0; i < b.N; i++ {
				token, err := benchClaims.ECDSASign(test.alg, test.key)
				if err != nil {
					b.Fatal(err)
				}
				tokenLen += len(token)
			}
			b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
		})
	}

	for _, test := range tests {
		token, err := benchClaims.ECDSASign(test.alg, test.key)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("check-"+test.alg, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := ECDSACheck(token, &test.key.PublicKey)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEdDSA(b *testing.B) {
	b.Run("sign-"+EdDSA, func(b *testing.B) {
		var tokenLen int
		for i := 0; i < b.N; i++ {
			token, err := benchClaims.EdDSASign(testKeyEd25519Private)
			if err != nil {
				b.Fatal(err)
			}
			tokenLen += len(token)
		}
		b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
	})

	b.Run("check-"+EdDSA, func(b *testing.B) {
		token, err := benchClaims.EdDSASign(testKeyEd25519Private)
		if err != nil {
			b.Fatal(err)
		}

		for i := 0; i < b.N; i++ {
			_, err := EdDSACheck(token, testKeyEd25519Public)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	var tokenString string

	privKey, err := dgrijalva.ParseEdPrivateKeyFromPEM([]byte(`-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIEFMEZrmlYxczXKFxIlNvNGR5JQvDhTkLovJYxwQd3ua
-----END PRIVATE KEY-----`))
	if err != nil {
		b.Fatal(err)
	}
	b.Run("sign-"+EdDSA+"-dgrijalva", func(b *testing.B) {
		method := dgrijalva.GetSigningMethod("EdDSA")
		var tokenLen int
		for i := 0; i < b.N; i++ {
			tokenString, err = dgrijalva.NewWithClaims(method, dgrijalvaClaims).SignedString(privKey)
			if err != nil {
				b.Fatal(err)
			}
			tokenLen += len(tokenString)
		}
		b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
	})

	pubKey, err := dgrijalva.ParseEdPublicKeyFromPEM([]byte(`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAWH7z6hpYqvPns2i4n9yymwvB3APhi4LyQ7iHOT6crtE=
-----END PUBLIC KEY-----`))
	if err != nil {
		b.Fatal(err)
	}
	b.Run("check-"+EdDSA+"-dgrijalva", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			t, err := dgrijalva.Parse(tokenString, func(token *dgrijalva.Token) (interface{}, error) {
				return pubKey, nil
			})
			if !t.Valid {
				b.Fatal(err)
			}
		}

	})
}

func BenchmarkHMAC(b *testing.B) {
	// 512-bit key
	secret := make([]byte, 64)
	algs := []string{HS256, HS384, HS512}

	for _, alg := range algs {
		b.Run("sign-"+alg, func(b *testing.B) {
			var tokenLen int
			for i := 0; i < b.N; i++ {
				token, err := benchClaims.HMACSign(alg, secret)
				if err != nil {
					b.Fatal(err)
				}
				tokenLen += len(token)
			}
			b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
		})

		b.Run("sign-"+alg+"-dgrijalva", func(b *testing.B) {
			method := dgrijalva.GetSigningMethod(alg)

			var tokenLen int
			for i := 0; i < b.N; i++ {
				token, err := dgrijalva.NewWithClaims(method, dgrijalvaClaims).SignedString(secret)
				if err != nil {
					b.Fatal(err)
				}
				tokenLen += len(token)
			}
			b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
		})
	}

	for _, alg := range algs {
		token, err := benchClaims.HMACSign(alg, secret)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("check-"+alg, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := HMACCheck(token, secret)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		tokenString := string(token)
		b.Run("check-"+alg+"-dgrijalva", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				t, err := dgrijalva.Parse(tokenString, func(token *dgrijalva.Token) (interface{}, error) {
					return secret, nil
				})
				if !t.Valid {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRSA(b *testing.B) {
	keys := []*rsa.PrivateKey{testKeyRSA1024, testKeyRSA2048, testKeyRSA4096}

	for _, key := range keys {
		b.Run(fmt.Sprintf("sign-%d-bit", key.Size()*8), func(b *testing.B) {
			var tokenLen int
			for i := 0; i < b.N; i++ {
				token, err := benchClaims.RSASign(RS384, key)
				if err != nil {
					b.Fatal(err)
				}
				tokenLen += len(token)
			}
			b.ReportMetric(float64(tokenLen)/float64(b.N), "B/token")
		})
	}

	for _, key := range keys {
		token, err := benchClaims.RSASign(RS384, key)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(fmt.Sprintf("check-%d-bit", key.Size()*8), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := RSACheck(token, &key.PublicKey)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
