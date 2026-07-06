package passkey

import (
	"fmt"
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/go-webauthn/webauthn/protocol"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
)

func BuildWebAuthn(r *http.Request) (*webauthn.WebAuthn, error) {
	common.OptionLock.RLock()
	enabled := common.OptionMap[model.OptionKeyPasskeyEnabled] == "true"
	rpID := strings.TrimSpace(common.OptionMap[model.OptionKeyPasskeyRPID])
	rpDisplayName := strings.TrimSpace(common.OptionMap[model.OptionKeyPasskeyRPDisplayName])
	originsText := strings.TrimSpace(common.OptionMap[model.OptionKeyPasskeyOrigins])
	allowInsecure := common.OptionMap[model.OptionKeyPasskeyAllowInsecure] == "true"
	userVerification := strings.TrimSpace(common.OptionMap[model.OptionKeyPasskeyUserVerification])
	attachment := strings.TrimSpace(common.OptionMap[model.OptionKeyPasskeyAttachment])
	common.OptionLock.RUnlock()

	if !enabled {
		return nil, fmt.Errorf("passkey 未启用")
	}
	if rpDisplayName == "" {
		rpDisplayName = common.SystemName
	}
	if rpID == "" {
		rpID = hostWithoutPort(r.Host)
	}
	// WebAuthn requires a domain, not an IP address
	if rpID == "127.0.0.1" || rpID == "0.0.0.0" {
		rpID = "localhost"
	}
	if rpID == "" {
		return nil, fmt.Errorf("passkey RPID 未配置")
	}

	origins := parseOrigins(originsText)
	if len(origins) == 0 {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		if scheme == "http" && !allowInsecure {
			return nil, fmt.Errorf("passkey 仅支持 HTTPS，请配置 passkey_allow_insecure=true 或使用 HTTPS")
		}
		origins = []string{fmt.Sprintf("%s://%s", scheme, r.Host)}
	}

	selection := protocol.AuthenticatorSelection{
		ResidentKey:        protocol.ResidentKeyRequirementRequired,
		RequireResidentKey: protocol.ResidentKeyRequired(),
		UserVerification:   protocol.UserVerificationRequirement(userVerification),
	}
	if selection.UserVerification == "" {
		selection.UserVerification = protocol.VerificationPreferred
	}
	if attachment != "" {
		selection.AuthenticatorAttachment = protocol.AuthenticatorAttachment(attachment)
	}

	config := &webauthn.Config{
		RPID:                   rpID,
		RPDisplayName:          rpDisplayName,
		RPOrigins:              origins,
		AuthenticatorSelection: selection,
		Debug:                  false,
	}
	return webauthn.New(config)
}

func parseOrigins(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		origins = append(origins, p)
	}
	return origins
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if i := strings.Index(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}
