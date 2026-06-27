// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package benchmark_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"go.xyrillian.de/gg/assert"
	pr "go.xyrillian.de/gg/pathrouter"
)

func BenchmarkRouterWithDistributionAPI(b *testing.B) {
	// This benchmark uses the OCI Distribution API as a moderately complex
	// real-world example of an HTTP API with parametrized endpoint paths.

	testWith := func(h http.Handler) func(b *testing.B) {
		return func(b *testing.B) {
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "/v2/foo/bar/manifests/latest", http.NoBody)
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				assert.Equal(b, rec.Code, http.StatusForbidden)
				assert.Equal(b, rec.Body.String(), "GetOrHeadManifest is forbidden\n")
				if b.Failed() {
					b.FailNow()
				}
			}
		}
	}

	b.Run("platform=gorilla-mux", testWith(buildMuxRouterForDistributionAPI()))
	b.Run("platform=gg-pathrouter", testWith(buildPathrouterForDistributionAPI()))
}

func buildMuxRouterForDistributionAPI() http.Handler {
	handle := func(endpointName string) http.HandlerFunc {
		msg := endpointName + " is forbidden"
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, msg, http.StatusForbidden)
		}
	}
	r := mux.NewRouter()
	r.Methods("GET").Path("/v2/").HandlerFunc(handle("GetToplevel"))
	r.Methods("GET").Path("/v2/_catalog").HandlerFunc(handle("GetCatalog"))
	r.Methods("DELETE").Path("/v2/{repository:.+}/blobs/{digest}").HandlerFunc(handle("DeleteBlob"))
	r.Methods("GET", "HEAD").Path("/v2/{repository:.+}/blobs/{digest}").HandlerFunc(handle("GetOrHeadBlob"))
	r.Methods("POST").Path("/v2/{repository:.+}/blobs/uploads/").HandlerFunc(handle("StartBlobUpload"))
	r.Methods("DELETE").Path("/v2/{repository:.+}/blobs/uploads/{uuid}").HandlerFunc(handle("DeleteBlobUpload"))
	r.Methods("GET").Path("/v2/{repository:.+}/blobs/uploads/{uuid}").HandlerFunc(handle("GetBlobUpload"))
	r.Methods("PATCH").Path("/v2/{repository:.+}/blobs/uploads/{uuid}").HandlerFunc(handle("ContinueBlobUpload"))
	r.Methods("PUT").Path("/v2/{repository:.+}/blobs/uploads/{uuid}").HandlerFunc(handle("FinishBlobUpload"))
	r.Methods("DELETE").Path("/v2/{repository:.+}/manifests/{reference}").HandlerFunc(handle("DeleteManifest"))
	r.Methods("GET", "HEAD").Path("/v2/{repository:.+}/manifests/{reference}").HandlerFunc(handle("GetOrHeadManifest"))
	r.Methods("PUT").Path("/v2/{repository:.+}/manifests/{reference}").HandlerFunc(handle("PutManifest"))
	r.Methods("GET").Path("/v2/{repository:.+}/referrers/{reference}").HandlerFunc(handle("GetReferrers"))
	r.Methods("GET").Path("/v2/{repository:.+}/tags/list").HandlerFunc(handle("ListTags"))
	return r
}

func buildPathrouterForDistributionAPI() http.Handler {
	handle := func(endpointName string) pr.HandlerFunc {
		msg := endpointName + " is forbidden"
		return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
			http.Error(w, msg, http.StatusForbidden)
		}
	}

	return pr.Choice(
		pr.Element("v2", pr.Choice(
			pr.Element("/", pr.Handlers(pr.ByMethod{
				http.MethodGet:  handle("GetToplevel"),
				http.MethodHead: nil,
			})),
			pr.Element("_catalog", pr.Handlers(pr.ByMethod{
				http.MethodGet:  handle("GetCatalog"),
				http.MethodHead: nil,
			})),
			pr.CatchAllVariable("repository", pr.Choice(
				pr.Element("blobs", pr.Choice(
					pr.Variable("digest", pr.Handlers(pr.ByMethod{
						http.MethodDelete: handle("DeleteBlob"),
						http.MethodGet:    handle("GetOrHeadBlob"),
					})),
					pr.Element("uploads", pr.Choice(
						pr.Element("/", pr.Handlers(pr.ByMethod{
							http.MethodPost: handle("StartBlobUpload"),
						})),
						pr.Variable("uuid", pr.Handlers(pr.ByMethod{
							http.MethodDelete: handle("DeleteBlobUpload"),
							http.MethodGet:    handle("GetBlobUpload"),
							http.MethodHead:   nil,
							http.MethodPatch:  handle("ContinueBlobUpload"),
							http.MethodPut:    handle("FinishBlobUpload"),
						})),
					)),
				)),
				pr.Element("manifests", pr.Variable("reference", pr.Handlers(pr.ByMethod{
					http.MethodDelete: handle("DeleteManifest"),
					http.MethodGet:    handle("GetOrHeadManifest"),
					http.MethodPut:    handle("PutManifest"),
				}))),
				pr.Element("referrers", pr.Variable("reference", pr.Handlers(pr.ByMethod{
					http.MethodGet:  handle("GetReferrers"),
					http.MethodHead: nil,
				}))),
				pr.Element("tags", pr.Element("list", pr.Handlers(pr.ByMethod{
					http.MethodGet:  handle("ListTags"),
					http.MethodHead: nil,
				}))),
			)),
		)),
	)
}
