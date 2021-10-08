package internal

import "testing"

func TestBasics(t *testing.T) {
	s := NewStore()

	// Make sure all the expected items are there.
	t.Run("list items", func(t *testing.T) {
		for _, it := range staticData {
			foundIt := s.Get(it.Code)
			if foundIt == nil || *foundIt != *it {
				t.Errorf("did not find expected item %s", it.Code)
			}
		}
	})

	newData := []*item{
		{Code: "0000-0000-0000-0000", Name: "Avocado", Price: 2.99},
		{Code: "MMMM-MMMM-MMMM-MMMM", Name: "Mango", Price: 1.59},
		{Code: "ZZZZ-ZZZZ-ZZZZ-ZZZZ", Name: "Zucchini", Price: 0.99},
	}

	// Make sure the new items are not yet there.
	t.Run("check missing items", func(t *testing.T) {
		for _, it := range newData {
			foundIt := s.Get(it.Code)
			if foundIt != nil {
				t.Errorf("found unexpected item %s", it.Code)
			}
		}
	})

	// Insert new items at the front, middle, back.
	for _, it := range newData {
		s.Put(*it)
	}

	// Make sure the new items are there now.
	t.Run("check new items", func(t *testing.T) {
		for _, it := range newData {
			foundIt := s.Get(it.Code)
			if foundIt == nil || *foundIt != *it {
				t.Errorf("did not find expected item %s", it.Code)
			}
		}
	})

	// List all the items, make sure they are sorted.
	t.Run("list items again", func(t *testing.T) {
		allItems, _ := s.List("", 100)
		if len(allItems) != 7 {
			t.Errorf("count want=%d, got=%d", 7, len(allItems))
		}
		var lastItem string
		for _, it := range allItems {
			t.Log(it.Code)
			if it.Code <= lastItem {
				t.Errorf("expected ascending: last=%s, this=%s", lastItem, it.Code)
			}
			lastItem = it.Code
		}
	})
}

func TestOverwrite(t *testing.T) {
	// Make sure overwrite works.
	s := NewStore()

	gotItem := s.Get("A12T-4GH7-QPL9-3N4M")
	if gotItem == nil || gotItem.Name != "Lettuce" || gotItem.Price != 3.46 {
		t.Errorf("wrong item")
	}

	// Lettuce went on sale; also it's now "romaine lettuce"
	s.Put(item{Code: "A12T-4GH7-QPL9-3N4M", Name: "Romaine Lettuce", Price: 3.00})

	// There should still only be 4 items.
	allItems, _ := s.List("", 100)
	if len(allItems) != 4 {
		t.Errorf("count want=%d, got=%d", 4, len(allItems))
	}

	gotItem = s.Get("A12T-4GH7-QPL9-3N4M")
	if gotItem == nil || gotItem.Name != "Romaine Lettuce" || gotItem.Price != 3.00 {
		t.Errorf("wrong item")
	}
}

func TestList(t *testing.T) {
	s := NewStore()

	tcs := []struct {
		cursor string
		count  int

		wantCount     int
		wantFirstItem string
		wantCursor    string
	}{
		// Starting from item 0.
		{"", 0, 0, "", ""},
		{"", 1, 1, "A12T-4GH7-QPL9-3N4M", "A12T-4GH7-QPL9-3N4M"},
		{"", 2, 2, "A12T-4GH7-QPL9-3N4M", "E5T6-9UI3-TH15-QR88"},
		{"", 3, 3, "A12T-4GH7-QPL9-3N4M", "TQ4C-VV6T-75ZX-1RMR"},
		{"", 4, 4, "A12T-4GH7-QPL9-3N4M", ""},
		{"", 5, 4, "A12T-4GH7-QPL9-3N4M", ""},

		// Starting from item 1.
		{"A12T-4GH7-QPL9-3N4M", 0, 0, "", "A12T-4GH7-QPL9-3N4M"},
		{"A12T-4GH7-QPL9-3N4M", 1, 1, "E5T6-9UI3-TH15-QR88", "E5T6-9UI3-TH15-QR88"},
		{"A12T-4GH7-QPL9-3N4M", 2, 2, "E5T6-9UI3-TH15-QR88", "TQ4C-VV6T-75ZX-1RMR"},
		{"A12T-4GH7-QPL9-3N4M", 3, 3, "E5T6-9UI3-TH15-QR88", ""},
		{"A12T-4GH7-QPL9-3N4M", 4, 3, "E5T6-9UI3-TH15-QR88", ""},

		// Starting from item 3.
		{"TQ4C-VV6T-75ZX-1RMR", 0, 0, "", "TQ4C-VV6T-75ZX-1RMR"},
		{"TQ4C-VV6T-75ZX-1RMR", 1, 1, "YRT6-72AS-K736-L4AR", ""},
		{"TQ4C-VV6T-75ZX-1RMR", 2, 1, "YRT6-72AS-K736-L4AR", ""},

		// Starting beyond the end.
		{"ZZZZ-ZZZZ-ZZZZ-ZZZZ", 0, 0, "", "ZZZZ-ZZZZ-ZZZZ-ZZZZ"},
		{"ZZZZ-ZZZZ-ZZZZ-ZZZZ", 1, 0, "", ""},
	}

	for i, tc := range tcs {
		ret, newCursor := s.List(tc.cursor, tc.count)
		if tc.wantCount != len(ret) {
			t.Errorf("case %d: count want=%d, got=%d", i, tc.wantCount, len(ret))
			continue
		}
		if tc.wantCount > 0 && tc.wantFirstItem != ret[0].Code {
			t.Errorf("case %d: first items want=%s, got=%s", i, tc.wantFirstItem, ret[0].Code)
			continue
		}
		if tc.wantCursor != newCursor {
			t.Errorf("case %d: cursor want=%s, got=%s", i, tc.wantCursor, newCursor)
			continue
		}
	}
}
