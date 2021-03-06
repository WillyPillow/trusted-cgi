package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/reddec/trusted-cgi/core/lambdas"
	"github.com/reddec/trusted-cgi/core/policy"
	"github.com/reddec/trusted-cgi/core/queue"
)

type Router struct {
	lambdaStorage lambdas.Storage
	policyStorage policy.Storage
	queue         queue.Queue
}

func (router *Router) Sync() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		router.routeRequest(writer, request, true)
	})
}

func (router *Router) Async() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		router.routeRequest(writer, request, false)
	})
}

func (router *Router) routeRequest(res http.ResponseWriter, req *http.Request, sync bool) {
	// TODO: add log everywhere
	uid := router.getLambdaUID(req)

	lambda, err := router.lambdaStorage.Find(uid)
	if errors.Is(err, lambdas.ErrNotFound) {
		http.NotFound(res, req)
		return
	}
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !router.isAllowed(uid, res, req) {
		return
	}

	if sync {
		lambda.ServeHTTP(res, req)
		return
	}

	correlationID, err := router.queue.Enqueue(req)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("X-Correlation-Id", correlationID)
	res.WriteHeader(http.StatusAccepted)
	_, _ = res.Write([]byte(correlationID))
}

func (router *Router) isAllowed(lambdaUID string, res http.ResponseWriter, req *http.Request) bool {
	assignedPolicy, err := router.policyStorage.FindByLambda(lambdaUID)

	switch {
	case errors.Is(err, policy.ErrNotFound): // no policy assigned - do nothing
	case err != nil:
		res.WriteHeader(http.StatusInternalServerError)
		return false
	case !assignedPolicy.IsAllowed(req): // prohibited
		res.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (router *Router) getLambdaUID(req *http.Request) string {
	path := req.URL.Path
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx:]
}
