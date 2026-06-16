package security

import (
	"encoding/json"
	"net/http"
)

type paymentRequiredResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SaaSValidationMiddleware intercepta requisições de estabelecimentos e valida
// assinatura SaaS ativa antes de liberar o fluxo.
func SaaSValidationMiddleware(guard *SaaSGuard) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			establishmentID, ok := EstablishmentIDFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "missing_establishment_context",
				})
				return
			}

			if err := guard.IsSubscriptionActive(r.Context(), establishmentID); err != nil {
				writePaymentRequired(w)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}

func writePaymentRequired(w http.ResponseWriter) {
	writeJSON(w, http.StatusPaymentRequired, paymentRequiredResponse{
		Error:   "subscription_expired_or_suspended",
		Message: "Assinatura vencida ou suspensa. Efetue o pagamento da mensalidade para liberar o acesso ao sistema.",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
