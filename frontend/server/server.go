// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package frontend

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/MelBookeen/readium-lcp-server/api"
	"github.com/MelBookeen/readium-lcp-server/frontend/api"
	"github.com/MelBookeen/readium-lcp-server/frontend/webpublication"
	"github.com/MelBookeen/readium-lcp-server/frontend/webpurchase"
	"github.com/MelBookeen/readium-lcp-server/frontend/webrepository"
	"github.com/MelBookeen/readium-lcp-server/frontend/webuser"
)

//Server struct contains server info and  db interfaces
type Server struct {
	http.Server
	readonly     bool
	cert         *tls.Certificate
	repositories webrepository.WebRepository
	publications webpublication.WebPublication
	users        webuser.WebUser
	purchases    webpurchase.WebPurchase
}

// HandlerFunc type define a function handled by the server
type HandlerFunc func(w http.ResponseWriter, r *http.Request, s staticapi.IServer)

//type HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s staticapi.IServer)

// New creates a new webserver (basic user interface)
func New(
	bindAddr string,
	tplPath string,
	repositoryAPI webrepository.WebRepository,
	publicationAPI webpublication.WebPublication,
	userAPI webuser.WebUser,
	purchaseAPI webpurchase.WebPurchase) *Server {

	sr := api.CreateServerRouter(tplPath)
	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           bindAddr,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		repositories: repositoryAPI,
		publications: publicationAPI,
		users:        userAPI,
		purchases:    purchaseAPI}

	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	apiURLPrefix := "/api/v1"

	//
	// repositories
	//
	repositoriesRoutesPathPrefix := apiURLPrefix + "/repositories"
	repositoriesRoutes := sr.R.PathPrefix(repositoriesRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	s.handleFunc(repositoriesRoutes, "/master-files", staticapi.GetRepositoryMasterFiles).Methods("GET")

	//
	// publications
	//
	publicationsRoutesPathPrefix := apiURLPrefix + "/publications"
	publicationsRoutes := sr.R.PathPrefix(publicationsRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	s.handleFunc(sr.R, publicationsRoutesPathPrefix, staticapi.GetPublications).Methods("GET")
	//
	s.handleFunc(sr.R, publicationsRoutesPathPrefix, staticapi.CreatePublication).Methods("POST")
	//
	s.handleFunc(publicationsRoutes, "/{id}", staticapi.GetPublication).Methods("GET")
	s.handleFunc(publicationsRoutes, "/{id}", staticapi.UpdatePublication).Methods("PUT")
	s.handleFunc(publicationsRoutes, "/{id}", staticapi.DeletePublication).Methods("DELETE")

	//
	// user functions
	//
	usersRoutesPathPrefix := apiURLPrefix + "/users"
	usersRoutes := sr.R.PathPrefix(usersRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	s.handleFunc(sr.R, usersRoutesPathPrefix, staticapi.GetUsers).Methods("GET")
	//
	s.handleFunc(sr.R, usersRoutesPathPrefix, staticapi.CreateUser).Methods("POST")
	//
	s.handleFunc(usersRoutes, "/{id}", staticapi.GetUser).Methods("GET")
	s.handleFunc(usersRoutes, "/{id}", staticapi.UpdateUser).Methods("PUT")
	s.handleFunc(usersRoutes, "/{id}", staticapi.DeleteUser).Methods("DELETE")
	//
	s.handleFunc(usersRoutes, "/{user_id}/purchases", staticapi.GetUserPurchases).Methods("GET")

	//
	// purchases
	//
	purchasesRoutesPathPrefix := apiURLPrefix + "/purchases"
	purchasesRoutes := sr.R.PathPrefix(purchasesRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	s.handleFunc(sr.R, purchasesRoutesPathPrefix, staticapi.GetPurchases).Methods("GET")
	//
	s.handleFunc(sr.R, purchasesRoutesPathPrefix, staticapi.CreatePurchase).Methods("POST")
	//
	s.handleFunc(purchasesRoutes, "/{id}", staticapi.GetPurchase).Methods("GET")
	s.handleFunc(purchasesRoutes, "/{id}", staticapi.UpdatePurchase).Methods("PUT")
	//
	s.handleFunc(purchasesRoutes, "/{id}/license", staticapi.GetPurchaseLicense).Methods("GET")
	//
	s.handleFunc(purchasesRoutes, "/license/{licenseID}", staticapi.GetPurchaseLicenseFromLicenseUUID).Methods("GET")

	return s
}

// RepositoryAPI ( staticapi.IServer ) returns interface for repositories
func (server *Server) RepositoryAPI() webrepository.WebRepository {
	return server.repositories
}

// PublicationAPI ( staticapi.IServer )returns DB interface for users
func (server *Server) PublicationAPI() webpublication.WebPublication {
	return server.publications
}

//UserAPI ( staticapi.IServer )returns DB interface for users
func (server *Server) UserAPI() webuser.WebUser {
	return server.users
}

//PurchaseAPI ( staticapi.IServer )returns DB interface for pruchases
func (server *Server) PurchaseAPI() webpurchase.WebPurchase {
	return server.purchases
}

func (server *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, server)
	})
}

/*no private functions used
func (server *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if api.CheckAuth(authenticator, w, r) {
			fn(w, r, server)
		}
	})
}
*/
