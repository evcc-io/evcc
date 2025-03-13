package vwidentity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	b := `<script>
        window._IDK = {
            templateModel: {"clientLegalEntityModel":{"clientId":"foo@apps_vw-dilab_com","clientAppName":"myAudi App","clientAppDisplayName":"myAudi App","legalEntityInfo":{"name":"Audi","shortName":"AUDI","productName":"Audi ID","theme":"audi","defaultLanguage":"de","termAndConditionsType":"DEFAULT","legalProperties":{"revokeDataContact":"","imprintText":"IMPRINT","countryOfJurisdiction":"DE"}},"informalLanguage":false,"legalEntityCode":"audi","imprintTextKey":"imprint.link.text"},"template":"loginAuthenticate","hmac":"16563a265369fbc1ef1ba126411349a430b1d8b5c1592b6284216dc634169ca3","useClientRendering":true,"titleKey":"title.login.password","title":null,"emailPasswordForm":{"@class":"com.volkswagen.identitykit.signin.domain.model.dto.EmailPasswordForm","email":"foo@bar","password":null},"error":null,"relayState":"be82f61e811860f3b4d8a78365253705bbdeaed4","nextButtonDisabled":false,"enableNextButtonAfterSeconds":0,"postAction":"login/authenticate","identifierUrl":"login/identifier"},
            disabledFeatures: {
                isRTLEnabled: false,
            },
            currentLocale: 'de-DE',
            csrf_parameterName: '_csrf',
            csrf_token: 'pwAX6CTKoSQWsej-9sROQ5LPw8Qy0gCT9oSdFxFxo9VkyNeYxjMg2kevkhY70Iyawul6e_D27qYK5za-leCoJXMSxu1R8eep',
            userSession: {
                userId: '',
                countryOfResidence: ''
            },
            baseUrl: 'https://identity.vwgroup.io',
            consentBaseUrl: 'http://consentapi-service.idk-peu.svc.cluster.local:8080',
            cancelContractUrl: ''
        }
    </script>`

	res, err := parseCredentials(b)
	require.NoError(t, err)
	require.Equal(t, "16563a265369fbc1ef1ba126411349a430b1d8b5c1592b6284216dc634169ca3", res.TemplateModel.Hmac)
	require.Equal(t, "pwAX6CTKoSQWsej-9sROQ5LPw8Qy0gCT9oSdFxFxo9VkyNeYxjMg2kevkhY70Iyawul6e_D27qYK5za-leCoJXMSxu1R8eep", res.CsrfToken)
}
