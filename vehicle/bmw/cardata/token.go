package cardata

import "golang.org/x/oauth2"

type Token struct {
	*oauth2.Token
	IdToken string `json:"id_token"`
	Gcid    string `json:"gcid"`
}

func (t *Token) TokenEx() *oauth2.Token {
	return t.Token.WithExtra(map[string]any{
		"id_token": t.IdToken,
		"gcid":     t.Gcid,
	})
}

type PersistingTokenSource struct {
	oauth2.TokenSource
	token   *oauth2.Token
	Persist func(token *oauth2.Token) error
}

func (pts *PersistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := pts.TokenSource.Token()
	if err != nil || token == pts.token {
		return token, err
	}
	if err := pts.Persist(token); err != nil {
		return nil, err
	}
	pts.token = token
	return token, nil
}

var TokenExtra = tokenExtra

func tokenExtra(t *oauth2.Token, key string) string {
	if v := t.Extra(key); v != nil {
		return v.(string)
	}
	return ""
}
