package emporia

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"golang.org/x/oauth2"
)

// nHex is the 3072-bit prime N used in the SRP protocol
const nHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6BF12FFA06D98A0864D87602733EC86A64521F2B18177B200CBBE117577A615D6C770988C0BAD946E208E24FA074E5AB3143DB5BFCE0FD108E4B82D120A93AD2CAFFFFFFFFFFFFFFFF"

const infoBits = "Caldera Derived Key"

// srpSession holds the SRP session values for Cognito authentication
type srpSession struct {
	n *big.Int
	g *big.Int
	k *big.Int
	a *big.Int
	A *big.Int
}

// newSRPSession creates a new SRP session with random values
func newSRPSession() (*srpSession, error) {
	n := new(big.Int)
	n.SetString(nHex, 16)

	g := big.NewInt(2)

	// k = SHA256(pad(N) | pad(g))
	nBytes, _ := hex.DecodeString(padHex(nHex))
	gBytes, _ := hex.DecodeString(padHex(fmt.Sprintf("%x", g)))
	kHash := sha256sum(append(nBytes, gBytes...))
	k := new(big.Int).SetBytes(kHash)

	// Generate random 'a'
	aBytes := make([]byte, 32)
	if _, err := rand.Read(aBytes); err != nil {
		return nil, fmt.Errorf("generate random a: %w", err)
	}
	a := new(big.Int).SetBytes(aBytes)

	// A = g^a mod N
	A := new(big.Int).Exp(g, a, n)

	return &srpSession{n: n, g: g, k: k, a: a, A: A}, nil
}

// padHex pads a hex string so it has an even number of characters and the high bit is not set
func padHex(hexStr string) string {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	if len(hexStr) >= 2 && hexStr[0] >= '8' {
		hexStr = "00" + hexStr
	}
	return hexStr
}

// sha256sum returns the SHA-256 hash of the input bytes
func sha256sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// hexHash computes SHA-256 of the bytes represented by a hex string and returns the result as hex
func hexHash(hexStr string) string {
	b, _ := hex.DecodeString(hexStr)
	return hex.EncodeToString(sha256sum(b))
}

// computeHKDF computes the HKDF key derivation function used in Cognito SRP.
// Note: 'ikm' is the S value, 'salt' is the u value (matching warrant/pycognito convention).
func computeHKDF(ikm, salt []byte) []byte {
	mac := hmac.New(sha256.New, salt)
	mac.Write(ikm)
	prk := mac.Sum(nil)

	info := append([]byte(infoBits), byte(1))
	mac2 := hmac.New(sha256.New, prk)
	mac2.Write(info)
	return mac2.Sum(nil)[:16]
}

// passwordAuthKey computes the HKDF authentication key using the SRP protocol
func (s *srpSession) passwordAuthKey(userID, password, bHex, saltHex, poolName string) ([]byte, error) {
	B := new(big.Int)
	B.SetString(bHex, 16)

	if new(big.Int).Mod(B, s.n).Sign() == 0 {
		return nil, fmt.Errorf("B mod N is zero")
	}

	// u = SHA256(pad(A) | pad(B))
	uHashHex := hexHash(padHex(fmt.Sprintf("%x", s.A)) + padHex(bHex))
	u := new(big.Int)
	u.SetString(uHashHex, 16)
	if u.Sign() == 0 {
		return nil, fmt.Errorf("u is zero")
	}

	// x = SHA256(salt_bytes | SHA256(poolName + userID + ":" + password))
	usernamePasswordHash := hex.EncodeToString(sha256sum([]byte(poolName + userID + ":" + password)))
	xHashHex := hexHash(padHex(saltHex) + usernamePasswordHash)
	x := new(big.Int)
	x.SetString(xHashHex, 16)

	// S = (B - k * g^x)^(a + u * x) mod N
	gx := new(big.Int).Exp(s.g, x, s.n)
	kgx := new(big.Int).Mul(s.k, gx)
	kgx.Mod(kgx, s.n)

	base := new(big.Int).Sub(B, kgx)
	base.Mod(base, s.n)
	if base.Sign() < 0 {
		base.Add(base, s.n)
	}

	exp := new(big.Int).Add(s.a, new(big.Int).Mul(u, x))
	S := new(big.Int).Exp(base, exp, s.n)

	sPadHex := padHex(fmt.Sprintf("%x", S))
	uPadHex := padHex(fmt.Sprintf("%x", u))

	sBytes, _ := hex.DecodeString(sPadHex)
	uBytes, _ := hex.DecodeString(uPadHex)

	return computeHKDF(sBytes, uBytes), nil
}

// challengeResponse computes the SRP challenge response for Cognito
func (s *srpSession) challengeResponse(userID, password, bHex, saltHex, secretBlock string) (map[string]string, error) {
	// poolName is the short form of the pool ID (part after the '_')
	poolName := strings.SplitN(UserPool, "_", 2)[1]

	hkdf, err := s.passwordAuthKey(userID, password, bHex, saltHex, poolName)
	if err != nil {
		return nil, err
	}

	secretBlockBytes, err := base64.StdEncoding.DecodeString(secretBlock)
	if err != nil {
		return nil, fmt.Errorf("decode secret block: %w", err)
	}

	// Timestamp format matches Python's: "Mon Jan  1 00:00:00 UTC 2024"
	timestamp := time.Now().UTC().Format("Mon Jan _2 15:04:05 UTC 2006")

	// HMAC-SHA256(hkdf, poolName | userID | secretBlock | timestamp)
	msg := []byte(poolName)
	msg = append(msg, []byte(userID)...)
	msg = append(msg, secretBlockBytes...)
	msg = append(msg, []byte(timestamp)...)

	mac := hmac.New(sha256.New, hkdf)
	mac.Write(msg)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"TIMESTAMP":                   timestamp,
		"USERNAME":                    userID,
		"PASSWORD_CLAIM_SECRET_BLOCK": secretBlock,
		"PASSWORD_CLAIM_SIGNATURE":    signature,
	}, nil
}

// Identity handles AWS Cognito authentication for Emporia
type Identity struct {
	client   *cognito.Client
	user     string
	password string
}

// NewIdentity creates a new Cognito identity client
func NewIdentity(user, password string) (*Identity, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(AWSRegion),
		awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Identity{
		client:   cognito.NewFromConfig(cfg),
		user:     user,
		password: password,
	}, nil
}

// Login authenticates with Cognito using SRP and returns an oauth2.Token.
// The AccessToken field contains the Cognito IdToken, which Emporia uses for API authorization.
func (id *Identity) Login() (*oauth2.Token, error) {
	return id.authenticate()
}

// Refresh uses the refresh token to obtain new tokens
func (id *Identity) Refresh(token *oauth2.Token) (*oauth2.Token, error) {
	ctx := context.Background()

	res, err := id.client.InitiateAuth(ctx, &cognito.InitiateAuthInput{
		AuthFlow: cotypes.AuthFlowTypeRefreshTokenAuth,
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": token.RefreshToken,
		},
		ClientId: aws.String(ClientID),
	})
	if err != nil {
		// Fall back to full re-authentication
		return id.authenticate()
	}

	result := res.AuthenticationResult
	if result == nil {
		return id.authenticate()
	}

	expiry := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	return &oauth2.Token{
		AccessToken:  aws.ToString(result.IdToken),
		RefreshToken: token.RefreshToken,
		Expiry:       expiry,
	}, nil
}

// authenticate performs full SRP authentication
func (id *Identity) authenticate() (*oauth2.Token, error) {
	ctx := context.Background()

	s, err := newSRPSession()
	if err != nil {
		return nil, fmt.Errorf("create srp session: %w", err)
	}

	// Step 1: InitiateAuth with SRP_A
	initRes, err := id.client.InitiateAuth(ctx, &cognito.InitiateAuthInput{
		AuthFlow: cotypes.AuthFlowTypeUserSrpAuth,
		AuthParameters: map[string]string{
			"USERNAME": id.user,
			"SRP_A":    fmt.Sprintf("%x", s.A),
		},
		ClientId: aws.String(ClientID),
	})
	if err != nil {
		return nil, fmt.Errorf("initiate auth: %w", err)
	}

	if initRes.ChallengeName != cotypes.ChallengeNameTypePasswordVerifier {
		return nil, fmt.Errorf("unexpected challenge: %s", initRes.ChallengeName)
	}

	params := initRes.ChallengeParameters
	userID := params["USERNAME"]
	bHex := params["SRP_B"]
	saltHex := params["SALT"]
	secretBlock := params["SECRET_BLOCK"]

	// Step 2: Compute challenge response
	resp, err := s.challengeResponse(userID, id.password, bHex, saltHex, secretBlock)
	if err != nil {
		return nil, fmt.Errorf("compute challenge response: %w", err)
	}

	// Step 3: RespondToAuthChallenge
	authRes, err := id.client.RespondToAuthChallenge(ctx, &cognito.RespondToAuthChallengeInput{
		ChallengeName:      cotypes.ChallengeNameTypePasswordVerifier,
		ClientId:           aws.String(ClientID),
		ChallengeResponses: resp,
	})
	if err != nil {
		return nil, fmt.Errorf("respond to auth challenge: %w", err)
	}

	result := authRes.AuthenticationResult
	if result == nil {
		return nil, fmt.Errorf("no authentication result")
	}

	expiry := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	// Emporia uses the IdToken (not AccessToken) for API authorization
	return &oauth2.Token{
		AccessToken:  aws.ToString(result.IdToken),
		RefreshToken: aws.ToString(result.RefreshToken),
		Expiry:       expiry,
	}, nil
}
