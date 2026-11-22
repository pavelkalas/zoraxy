package auth

import (
	"errors"
	"fmt"
	"net/http"
)

type RouterOption struct {
	AuthAgent     *AuthAgent
	RequireAuth   bool                                     //This router require authentication
	DeniedHandler func(http.ResponseWriter, *http.Request) //Things to do when request is rejected
	TargetMux     *http.ServeMux
}

type RouterDef struct {
	option    RouterOption
	endpoints map[string]func(http.ResponseWriter, *http.Request)
}

func NewManagedHTTPRouter(option RouterOption) *RouterDef {
	return &RouterDef{
		option:    option,
		endpoints: map[string]func(http.ResponseWriter, *http.Request){},
	}
}

// denyWithJSON pošle klientovi JSON chybu se statusem 403.
func denyWithJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("{\"error\":\"" + message + "\"}"))
}

func (router *RouterDef) HandleFunc(endpoint string, handler func(http.ResponseWriter, *http.Request)) error {
	//Check if the endpoint already registered
	if _, exist := router.endpoints[endpoint]; exist {
		fmt.Println("WARNING! Duplicated registering of web endpoint: " + endpoint)
		return errors.New("endpoint register duplicated")
	}

	authAgent := router.option.AuthAgent

	//Obal handleru, který po úspěšném přihlášení ještě zkontroluje
	//povinnou změnu hesla a oprávnění pro daný endpoint.
	authenticatedHandler := func(w http.ResponseWriter, r *http.Request) {
		//Pokud si uživatel ještě nezměnil heslo, nesmí na nic jiného než na
		//endpointy pro změnu hesla / odhlášení.
		if authAgent.MustChangePassword(r) && !authAgent.IsPasswordChangeAllowedEndpoint(endpoint) {
			denyWithJSON(w, "password change required")
			return
		}

		permission := authAgent.PermissionForPath(endpoint)
		if permission == "" {
			//Neznámý endpoint: dostane se k němu jen admin.
			if !authAgent.IsAdmin(r) {
				denyWithJSON(w, "insufficient permission")
				return
			}
		} else if !authAgent.UserHasPermission(r, permission) {
			denyWithJSON(w, "insufficient permission")
			return
		}

		handler(w, r)
	}

	//OK. Register handler
	if router.option.TargetMux == nil {
		http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
			//Check authentication of the user
			if router.option.RequireAuth {
				authAgent.HandleCheckAuth(w, r, authenticatedHandler)
			} else {
				handler(w, r)
			}

		})
	} else {
		router.option.TargetMux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
			//Check authentication of the user
			if router.option.RequireAuth {
				authAgent.HandleCheckAuth(w, r, authenticatedHandler)
			} else {
				handler(w, r)
			}

		})
	}

	router.endpoints[endpoint] = handler

	return nil
}
