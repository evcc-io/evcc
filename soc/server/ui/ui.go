package ui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andig/evcc/soc/server/auth"
	"github.com/andig/evcc/util"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

//go:embed index.html
var indexHtml string

//go:embed privacy.html
var privacyHtml string

var (
	// login ui and callback
	redirectURL = util.Getenv("REDIRECT_URL")

	// SSL termination
	sslCertDir = util.Getenv("SSL_CERT_DIR", "certs")

	// github oauth application
	clientID     = util.Getenv("GITHUB_CLIENT_ID")
	clientSecret = util.Getenv("GITHUB_CLIENT_SECRET")
)

var (
	indexTpl, privacyTpl *template.Template

	oauthState  = randomString()
	oauthConfig *oauth2.Config
)

const (
	ProfileURL = "https://api.github.com/user"
	EmailURL   = "https://api.github.com/user/emails"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
	Name     string `json:"name"`
	Login    string `json:"login"`
	Picture  string `json:"avatar_url"`
	Location string `json:"location"`
}

func init() {
	clientID = util.Getenv("GITHUB_CLIENT_ID")
	clientSecret = util.Getenv("GITHUB_CLIENT_SECRET")

	oauthConfig = &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	}

	indexTpl = template.Must(template.New("index").Parse(indexHtml))
	privacyTpl = template.Must(template.New("privacy").Parse(privacyHtml))
}

func randomString() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	length := 16
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	_ = indexTpl.Execute(w, nil)
}

func handlePrivacy(w http.ResponseWriter, r *http.Request) {
	_ = privacyTpl.Execute(w, nil)
}

func templateError(w http.ResponseWriter, r *http.Request, err string) {
	_ = indexTpl.Execute(w, map[string]interface{}{
		"Content": template.HTML(`
<h1>Fehler</h1>
<p class="lead">` + err + `</p>`),
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("error") != "" {
		err := r.URL.Query().Get("error_description")
		templateError(w, r, err)
		return
	}

	user, err := getUserInfo(r.FormValue("state"), r.FormValue("code"))
	if err != nil {
		templateError(w, r, err.Error())
		return
	}

	authorized, err := auth.IsSponsor(user.Login)
	if err != nil {
		templateError(w, r, err.Error())
		return
	}

	authMap := map[bool]string{false: "rejected", true: "authorized"}
	log.Println(user.Login+":", authMap[authorized])

	if !authorized {
		_ = indexTpl.Execute(w, map[string]interface{}{
			"Content": template.HTML(`
	<h1>Inaktiv</h1>
	<p class="lead">Github Sponsorship für evcc ist inaktiv. Um erweiterte Funktionen zu nutzen, ist es notwendig, evcc zu unterstützen.</p>
	<p class="lead">evcc bei Github <a href="https://github.com/sponsors/andig" class="text-warning">unterstützen</a>.</p>`),
		})
		return
	}

	claims := auth.Claims{
		Username: user.Name,
		StandardClaims: jwt.StandardClaims{
			Subject: user.Login,
		},
	}

	jwt, err := auth.AuthorizedToken(claims)
	if err != nil {
		templateError(w, r, err.Error())
		return
	}

	_ = indexTpl.Execute(w, map[string]interface{}{
		"Content": template.HTML(fmt.Sprintf(`
<h1>Aktiv</h1>
<p class="lead">Github Sponsorship für evcc ist aktiv. Das folgende Registrierungstoken kann für den Zugriff auf evcc genutzt werden.</p>
<p class="lead">Der Code ist %d Tage gültig und kann jederzeit neu erzeugt werden.</p>
<p class="lead"><code>`+jwt+`</code></p>`, auth.TokenExpiry),
		)})
}

func getUserInfo(state string, code string) (*User, error) {
	if state != oauthState {
		return nil, fmt.Errorf("invalid oauth state")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	var user User
	req, err := http.NewRequest("GET", ProfileURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API responded with a %d trying to fetch user information", response.StatusCode)
	}

	bits, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bits, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func Run() {
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleMain)
	mux.HandleFunc("/privacy", handlePrivacy)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/callback", handleCallback)

	s := &http.Server{
		Addr:    ":http",
		Handler: mux,
	}

	if sslHosts := strings.TrimSpace(os.Getenv("SSL_HOSTS")); sslHosts != "" {
		log.Println("sslHosts:", sslHosts)
		log.Println("sslCertDir:", sslCertDir)

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(strings.Split(sslHosts, " ")...),
			Cache:      autocert.DirCache(sslCertDir),
		}

		go func() {
			log.Fatal(http.ListenAndServe(":http", certManager.HTTPHandler(nil)))
		}()

		s.Addr = ":https"
		s.TLSConfig = certManager.TLSConfig()

		log.Fatal(s.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(s.ListenAndServe())
	}
}
