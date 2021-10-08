package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type errRsp struct {
	Error string
}

var (
	codeRegex = regexp.MustCompile("^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$")
)

type jsonItem struct {
	item
	Url string `json:"url"`
}

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{
		store: store,
	}
}

func (svc *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("/items", svc.HandleItems)
	mux.HandleFunc("/items/", svc.HandleItem)
}

// HandleItems handles /items
func (svc *Service) HandleItems(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	switch r.Method {
	case "GET", "HEAD":
		svc.HandleItemsList(w, r, enc)
	case "PUT", "POST":
		svc.HandleItemPut(w, r, "", enc)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleItem handles /items/:code
func (svc *Service) HandleItem(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	code := strings.TrimPrefix(r.URL.Path, "/items/")
	code = strings.ToUpper(code)
	if !codeRegex.MatchString(code) {
		w.WriteHeader(http.StatusBadRequest)
		_ = enc.Encode(errRsp{Error: "bad code format"})
		return
	}

	switch r.Method {
	case "GET", "HEAD":
		svc.HandleItemGet(w, r, code, enc)
	case "PUT", "POST":
		svc.HandleItemPut(w, r, code, enc)
	case "DELETE":
		// TODO: support deletion
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (svc *Service) HandleItemsList(w http.ResponseWriter, r *http.Request, enc *json.Encoder) {
	query := r.URL.Query()
	count := 10
	if query.Has("count") {
		var err error
		count, err = strconv.Atoi(query.Get("count"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = enc.Encode(errRsp{Error: "bad count: " + err.Error()})
			return
		}
	}

	cursor := strings.ToUpper(query.Get("cursor"))
	if cursor != "" && !codeRegex.MatchString(cursor) {
		w.WriteHeader(http.StatusBadRequest)
		_ = enc.Encode(errRsp{Error: "bad cursor"})
		return
	}

	items, nextCursor := svc.store.List(cursor, count)
	if nextCursor != "" {
		w.Header().Set("Link", fmt.Sprintf(`</items?count=%d&cursor=%s>; rel="next\"`, count, nextCursor))
	}

	// Fixup items to include a URL.
	jsonItems := make([]jsonItem, 0, len(items))
	for _, it := range items {
		jsonItems = append(jsonItems, jsonItem{
			item: it,
			Url:  "/items/" + it.Code,
		})
	}

	w.WriteHeader(http.StatusOK)
	_ = enc.Encode(jsonItems)
}

func (svc *Service) HandleItemGet(w http.ResponseWriter, r *http.Request, code string, enc *json.Encoder) {
	it := svc.store.Get(code)
	if it == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = enc.Encode(errRsp{Error: "no such item"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = enc.Encode(jsonItem{
		item: *it,
		Url:  "/items/" + it.Code,
	})
}

func (svc *Service) HandleItemPut(w http.ResponseWriter, r *http.Request, code string, enc *json.Encoder) {
	// Try to parse the body.
	var it item
	err := json.NewDecoder(r.Body).Decode(&it)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = enc.Encode(errRsp{Error: "could not parse payload: " + err.Error()})
		return
	}

	it.Code = strings.ToUpper(it.Code)

	// If the body did not contain a code, use the code from the URL.
	if it.Code == "" {
		it.Code = code
	}

	// If both were specified, they better match.
	if code != "" && code != it.Code {
		w.WriteHeader(http.StatusBadRequest)
		_ = enc.Encode(errRsp{Error: "code in url and body did not match"})
		return

	}

	if it.Code == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "missing code"})
		return
	}

	if !codeRegex.MatchString(it.Code) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "bad code format"})
		return
	}

	if it.Name == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "missing name"})
		return
	}

	switch {
	case it.Price == 0:
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "missing price"})
		return
	case it.Price < 0:
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "negative price"})
		return
	case it.Price > 9999:
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = enc.Encode(errRsp{Error: "price must be <= $10000"})
		return

	}

	svc.store.Put(it)

	// Respond with the new item.
	svc.HandleItemGet(w, r, it.Code, enc)
}
