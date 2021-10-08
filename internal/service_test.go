package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestService(t *testing.T) {
	store := NewStore()
	svc := &Service{store: store}
	mux := http.NewServeMux()
	svc.Register(mux)

	svr := httptest.NewServer(mux)
	defer svr.Close()

	t.Run("list items", func(t *testing.T) {
		rsp, err := http.Get(svr.URL + "/items")
		if err != nil {
			t.Fatal(err)
		}
		defer drainAndClose(rsp.Body)
		if rsp.StatusCode != http.StatusOK {
			t.Fatal(rsp.StatusCode)
		}
		var items []jsonItem
		err = json.NewDecoder(rsp.Body).Decode(&items)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 4 {
			t.Errorf("want 4 items, got: %d", len(items))
		}
	})

	t.Run("get a known items", func(t *testing.T) {
		rsp, err := http.Get(svr.URL + "/items/A12T-4GH7-QPL9-3N4M")
		if err != nil {
			t.Fatal(err)
		}
		defer drainAndClose(rsp.Body)

		if rsp.StatusCode != http.StatusOK {
			t.Fatal(rsp.StatusCode)
		}
		var it jsonItem
		err = json.NewDecoder(rsp.Body).Decode(&it)
		if err != nil {
			t.Fatal(err)
		}
		if it.Name != "Lettuce" {
			t.Errorf("want Lettuce, got: %s", it.Name)
		}
	})

	t.Run("bad request code", func(t *testing.T) {
		rsp, err := http.Get(svr.URL + "/items/whatever")
		if err != nil {
			t.Fatal(err)
		}
		defer drainAndClose(rsp.Body)

		if rsp.StatusCode != http.StatusBadRequest {
			t.Fatal(rsp.StatusCode)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rsp, err := http.Get(svr.URL + "/items/AAAA-AAAA-AAAA-AAAA")
		if err != nil {
			t.Fatal(err)
		}
		defer drainAndClose(rsp.Body)
		if rsp.StatusCode != http.StatusNotFound {
			t.Fatal(rsp.StatusCode)
		}
	})

	t.Run("create item check codes", func(t *testing.T) {
		tcs := []struct {
			code         string
			url          string
			expectStatus int
		}{
			{"AAAA-AAAA-AAAA-AAAA", "/items/AAAA-AAAA-AAAA-AAAA", http.StatusOK},         // both match ok
			{"AAAA-AAAA-AAAA-AAAA", "/items", http.StatusOK},                             // only in item
			{"", "/items/AAAA-AAAA-AAAA-AAAA", http.StatusOK},                            // only in url
			{"AAAA-AAAA-AAAA-AAAA", "/items/BBBB-BBBB-BBBB-BBBB", http.StatusBadRequest}, // mismatch bad
			{"AAAAAAAAAAAAAAAA", "/items", http.StatusUnprocessableEntity},               // bad format
			{"", "/items/AAAAAAAAAAAAAAAA", http.StatusBadRequest},                       // bad format
		}

		for i, tc := range tcs {
			var buf strings.Builder
			err := json.NewEncoder(&buf).Encode(item{
				Code:  tc.code,
				Name:  "Avacado",
				Price: 4.99,
			})
			if err != nil {
				t.Fatal(err)
			}

			rsp, err := http.Post(svr.URL+tc.url, "application/json", strings.NewReader(buf.String()))
			if err != nil {
				t.Fatal(err)
			}
			defer drainAndClose(rsp.Body)

			if rsp.StatusCode != tc.expectStatus {
				t.Fatalf("case %d: want=%d, got=%d", i, tc.expectStatus, rsp.StatusCode)
			}
			if tc.expectStatus == http.StatusOK {
				var it jsonItem
				err = json.NewDecoder(rsp.Body).Decode(&it)
				if err != nil {
					t.Fatal(err)
				}
				if it.Name != "Avacado" {
					t.Errorf("want Avacado, got: %s", it.Name)
				}
			}
		}
	})

	t.Run("create item check prices", func(t *testing.T) {
		tcs := []struct {
			price        float32
			expectStatus int
		}{
			{0, http.StatusUnprocessableEntity},
			{-2, http.StatusUnprocessableEntity},
			{10000, http.StatusUnprocessableEntity},
			{4.99, http.StatusOK},
		}

		for i, tc := range tcs {
			var buf strings.Builder
			err := json.NewEncoder(&buf).Encode(item{
				Name:  "Avacado",
				Price: tc.price,
			})
			if err != nil {
				t.Fatal(err)
			}

			rsp, err := http.Post(svr.URL+"/items/AAAA-AAAA-AAAA-AAAA", "application/json", strings.NewReader(buf.String()))
			if err != nil {
				t.Fatal(err)
			}
			defer drainAndClose(rsp.Body)

			if rsp.StatusCode != tc.expectStatus {
				t.Fatalf("case %d: want=%d, got=%d", i, tc.expectStatus, rsp.StatusCode)
			}
			if tc.expectStatus == http.StatusOK {
				var it jsonItem
				err = json.NewDecoder(rsp.Body).Decode(&it)
				if err != nil {
					t.Fatal(err)
				}
				if it.Name != "Avacado" {
					t.Errorf("want Avacado, got: %s", it.Name)
				}
			}
		}
	})

	t.Run("list items again", func(t *testing.T) {
		rsp, err := http.Get(svr.URL + "/items")
		if err != nil {
			t.Fatal(err)
		}
		if rsp.StatusCode != http.StatusOK {
			t.Fatal(rsp.StatusCode)
		}
		defer drainAndClose(rsp.Body)

		var items []jsonItem
		err = json.NewDecoder(rsp.Body).Decode(&items)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 5 {
			t.Errorf("want 5 items, got: %d", len(items))
		}
	})
}

func drainAndClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
}

func TestCodeRegex(t *testing.T) {
	tcs := []struct {
		input    string
		expectOk bool
	}{
		{"", false},
		{"AAAA-AAAA-AAAA-AAAA", true},
		{"AAAA-AAAA-AAAA-AAA", false},
		{"AAAA-AAAA-AAAA", false},
		{"AAAA-AAAA-AAAA-AAAA-AAAA", false},
		{"AAAAAAAAAAAAAAAA", false},
		{"1111-1111-1111-1111", true},
		{"$AAA-AAAA-AAAA-AAAA", false},
	}

	for _, tc := range tcs {
		got := codeRegex.MatchString(tc.input)
		if tc.expectOk != got {
			t.Errorf("want=%t, got=%t for input=%q", tc.expectOk, got, tc.input)
		}
	}
}
